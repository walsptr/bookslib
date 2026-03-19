pipeline {
  agent { label 'docker-host' }

  environment {
    GITOPS_REPO_URL = 'git@github.com:walsptr/gitops.git'
    GITOPS_BRANCH = 'main'

    RUN_TESTS = 'false'
    FORCE_BUILD_ALL = 'false'

    DOCKERHUB_USER = 'sywlsptr'

    AUTH_IMAGE = "${DOCKERHUB_USER}/auth-service"
    BOOKS_IMAGE = "${DOCKERHUB_USER}/books-service"
    REVIEWS_IMAGE = "${DOCKERHUB_USER}/reviews-service"
    FRONTEND_IMAGE = "${DOCKERHUB_USER}/bookslib-frontend"
  }

  options {
    timestamps()
    disableConcurrentBuilds()
  }

  stages {
    stage('Checkout') {
      steps {
        checkout(scm)
      }
    }

    stage('Compute Version') {
      steps {
        script {
          env.IMAGE_TAG = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
          env.COMMIT_SHA = sh(script: 'git rev-parse HEAD', returnStdout: true).trim()
        }
      }
    }

    stage('Detect Changed Services') {
      steps {
        script {
          def hasParent = sh(script: 'git rev-parse HEAD~1 >/dev/null 2>&1', returnStatus: true) == 0
          def diff = ''

          if (hasParent) {
            def base = sh(script: "git rev-parse HEAD~1", returnStdout: true).trim()
            diff = sh(script: "git diff --name-only ${base} ${env.COMMIT_SHA}", returnStdout: true).trim()
          }

          def changed = [
            auth: (diff.contains('auth-service/') || diff.contains('helm/auth-service/')),
            books: (diff.contains('books-service/') || diff.contains('helm/books-service/')),
            reviews: (diff.contains('reviews-service/') || diff.contains('helm/reviews-service/')),
            frontend: (diff.contains('frontend/') || diff.contains('helm/frontend/'))
          ]

          if (!diff) {
            changed = [auth: true, books: true, reviews: true, frontend: true]
          }

          env.CHANGE_AUTH = changed.auth.toString()
          env.CHANGE_BOOKS = changed.books.toString()
          env.CHANGE_REVIEWS = changed.reviews.toString()
          env.CHANGE_FRONTEND = changed.frontend.toString()
        }
      }
    }

    stage('Select Environment') {
      steps {
        script {
          def isPR = env.CHANGE_ID
          env.DEPLOY_ENABLED = (!isPR && (env.BRANCH_NAME == 'dev' || env.BRANCH_NAME == 'prod')) ? 'true' : 'false'

          env.BUILD_AUTH = env.CHANGE_AUTH
          env.BUILD_BOOKS = env.CHANGE_BOOKS
          env.BUILD_REVIEWS = env.CHANGE_REVIEWS
          env.BUILD_FRONTEND = env.CHANGE_FRONTEND

          if (env.DEPLOY_ENABLED != 'true') {
            env.DEPLOY_ENV = 'none'
            env.GITOPS_PATH_PREFIX = ''
            env.GITOPS_PUSH_BRANCH = ''
            return
          }

          if (env.BRANCH_NAME == 'dev') {
            env.DEPLOY_ENV = 'dev'
            env.GITOPS_PATH_PREFIX = 'environments/dev'
            env.GITOPS_PUSH_BRANCH = env.GITOPS_BRANCH
          } else {
            env.DEPLOY_ENV = 'prod'
            env.GITOPS_PATH_PREFIX = 'environments/prod'
            env.GITOPS_PUSH_BRANCH = "promote/${env.IMAGE_TAG}"
          }
        }
      }
    }

    stage('Decide Build Set') {
      when {
        expression { return env.DEPLOY_ENABLED == 'true' }
      }
      steps {
        script {
          def hasCurl = sh(script: 'command -v curl >/dev/null 2>&1', returnStatus: true) == 0

          def repoExists = { String repo ->
            if (!hasCurl) {
              return true
            }
            return sh(
              script: "curl -fsS https://hub.docker.com/v2/repositories/${env.DOCKERHUB_USER}/${repo}/ >/dev/null",
              returnStatus: true
            ) == 0
          }

          if (env.FORCE_BUILD_ALL == 'true') {
            env.BUILD_AUTH = 'true'
            env.BUILD_BOOKS = 'true'
            env.BUILD_REVIEWS = 'true'
            env.BUILD_FRONTEND = 'true'
          } else {
            env.BUILD_AUTH = (env.CHANGE_AUTH == 'true' || !repoExists('auth-service')) ? 'true' : 'false'
            env.BUILD_BOOKS = (env.CHANGE_BOOKS == 'true' || !repoExists('books-service')) ? 'true' : 'false'
            env.BUILD_REVIEWS = (env.CHANGE_REVIEWS == 'true' || !repoExists('reviews-service')) ? 'true' : 'false'
            env.BUILD_FRONTEND = (env.CHANGE_FRONTEND == 'true' || !repoExists('bookslib-frontend')) ? 'true' : 'false'
          }
        }
      }
    }

    stage('Registry Login') {
      when {
        expression { return env.DEPLOY_ENABLED == 'true' }
      }
      steps {
        withCredentials([usernamePassword(credentialsId: 'registry-creds', usernameVariable: 'REG_USER', passwordVariable: 'REG_PASS')]) {
          sh '''
            echo "$REG_PASS" | docker login -u "$REG_USER" --password-stdin
          '''
        }
      }
    }

    stage('Build & Test') {
      when {
        expression { return env.RUN_TESTS == 'true' }
      }
      steps {
        script {
          if (env.BUILD_AUTH == 'true') {
            sh 'docker build -f auth-service/Dockerfile --target test -t auth-service:test auth-service'
          }
          if (env.BUILD_BOOKS == 'true') {
            sh 'docker build -f books-service/Dockerfile --target test -t books-service:test books-service'
          }
          if (env.BUILD_REVIEWS == 'true') {
            sh 'docker build -f reviews-service/Dockerfile --target test -t reviews-service:test reviews-service'
          }
          if (env.BUILD_FRONTEND == 'true') {
            sh 'docker build -f frontend/Dockerfile --target test -t frontend:test frontend'
          }
        }
      }
    }

    stage('Build Production Images') {
      when {
        expression { return env.DEPLOY_ENABLED == 'true' }
      }
      steps {
        script {
          if (env.BUILD_AUTH == 'true') {
            sh "docker build -f auth-service/Dockerfile --target production -t ${AUTH_IMAGE}:${IMAGE_TAG} auth-service"
          }
          if (env.BUILD_BOOKS == 'true') {
            sh "docker build -f books-service/Dockerfile --target production -t ${BOOKS_IMAGE}:${IMAGE_TAG} books-service"
          }
          if (env.BUILD_REVIEWS == 'true') {
            sh "docker build -f reviews-service/Dockerfile --target production -t ${REVIEWS_IMAGE}:${IMAGE_TAG} reviews-service"
          }
          if (env.BUILD_FRONTEND == 'true') {
            sh "docker build -f frontend/Dockerfile --target production -t ${FRONTEND_IMAGE}:${IMAGE_TAG} frontend"
          }
        }
      }
    }

    stage('Push Images') {
      when {
        expression { return env.DEPLOY_ENABLED == 'true' }
      }
      steps {
        script {
          if (env.BUILD_AUTH == 'true') {
            sh "docker push ${AUTH_IMAGE}:${IMAGE_TAG}"
          }
          if (env.BUILD_BOOKS == 'true') {
            sh "docker push ${BOOKS_IMAGE}:${IMAGE_TAG}"
          }
          if (env.BUILD_REVIEWS == 'true') {
            sh "docker push ${REVIEWS_IMAGE}:${IMAGE_TAG}"
          }
          if (env.BUILD_FRONTEND == 'true') {
            sh "docker push ${FRONTEND_IMAGE}:${IMAGE_TAG}"
          }
        }
      }
    }

    stage('Checkout GitOps Repo') {
      when {
        expression { return env.DEPLOY_ENABLED == 'true' }
      }
      steps {
        dir('gitops') {
          checkout([
            $class: 'GitSCM',
            branches: [[name: "*/${GITOPS_BRANCH}"]],
            userRemoteConfigs: [[url: "${GITOPS_REPO_URL}", credentialsId: 'gitops-push-key']]
          ])
          sh 'git config user.email "jenkins@local"'
          sh 'git config user.name "jenkins"'
        }
      }
    }

    stage('Update GitOps Values') {
      when {
        expression { return env.DEPLOY_ENABLED == 'true' }
      }
      steps {
        dir('gitops') {
          script {
            def updateYaml = { filePath ->
              sh """
                python3 - << 'PY'
from pathlib import Path
import yaml

path = Path('${filePath}')
data = yaml.safe_load(path.read_text())
data.setdefault('image', {})
data['image']['tag'] = '${env.IMAGE_TAG}'
path.write_text(yaml.safe_dump(data, sort_keys=False))
PY
              """
            }

            if (env.BUILD_AUTH == 'true') updateYaml("${env.GITOPS_PATH_PREFIX}/auth-service.values.yaml")
            if (env.BUILD_BOOKS == 'true') updateYaml("${env.GITOPS_PATH_PREFIX}/books-service.values.yaml")
            if (env.BUILD_REVIEWS == 'true') updateYaml("${env.GITOPS_PATH_PREFIX}/reviews-service.values.yaml")
            if (env.BUILD_FRONTEND == 'true') updateYaml("${env.GITOPS_PATH_PREFIX}/frontend.values.yaml")
          }
        }
      }
    }

    stage('Commit & Push GitOps') {
      when {
        expression { return env.DEPLOY_ENABLED == 'true' }
      }
      steps {
        dir('gitops') {
          script {
            sh "git status"
            sh "git checkout -B ${env.GITOPS_PUSH_BRANCH}"
            sh "git add ${env.GITOPS_PATH_PREFIX}/*.values.yaml"
            sh "git commit -m \"${env.DEPLOY_ENV}: bump images to ${env.IMAGE_TAG}\" || true"
            sshagent(credentials: ['gitops-push-key']) {
              sh "git push origin ${env.GITOPS_PUSH_BRANCH}"
            }
          }
        }
      }
    }
  }

  post {
    always {
      sh 'docker logout || true'
    }
  }
}
