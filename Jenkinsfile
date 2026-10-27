pipeline {
    agent any

    options {
        timestamps()
        disableConcurrentBuilds()
        buildDiscarder(logRotator(numToKeepStr: '20'))
    }

    environment {
        COMPOSE_PROJECT_NAME = 'afroracle'
    }

    stages {
        stage('Checkout') {
            steps { checkout scm }
        }

        stage('Backend — test') {
            steps {
                dir('backend') {
                    sh 'go vet ./...'
                    sh 'go test ./...'
                    sh 'go build ./...'
                }
            }
        }

        stage('Frontend — build') {
            steps {
                dir('frontend') {
                    sh 'npm ci || npm install'
                    sh 'npm run build'
                }
            }
        }

        stage('Docker — build & smoke') {
            steps {
                sh 'docker compose build'
                sh 'docker compose up -d'
                // Backend must answer /api/state, then demo must still pass in mock mode.
                sh '''
                    for i in $(seq 1 30); do
                      curl -sf http://localhost:8000/api/state && break || sleep 2
                    done
                    curl -sf http://localhost:8000/api/state
                    for i in $(seq 1 30); do
                      curl -sf http://localhost:3000 >/dev/null && break || sleep 2
                    done
                    curl -sf http://localhost:3000 >/dev/null
                '''
            }
        }

        stage('Demo — mock contract') {
            steps {
                sh 'BASE=http://localhost:8000 bash scripts/demo.sh'
            }
        }
    }

    post {
        always {
            sh 'docker compose down -v || true'
        }
        failure {
            sh 'docker compose logs --tail=100 || true'
        }
    }
}
