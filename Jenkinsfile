pipeline {
    agent none

    environment {
        PROJECT_KEY = 'go-password-checker'
        SONAR_SERVER = 'sonar-server'
    }
    
    stages {
        stage('Build') {
            agent {
                docker {
                    image 'golang:1.26.2-alpine3.23'
                    reuseNode true
                }
            }
            steps {
                sh '''
                    go mod download

                    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -v ./...
                '''
            }
        }   
        stage('Test') {
            agent {
                docker {
                    image 'golang:1.26.2-alpine3.23'
                    reuseNode true
                }
            }
            steps {
                sh 'go test -v -coverprofile=coverage.out ./...'
            }
        }
        stage('SonarQube Analysis') {
            agent {
                docker {
                    image 'sonarsource/sonar-scanner-cli'
                    reuseNode true
                }
            }
            steps {
                withSonarQubeEnv(SONAR_SERVER) {
                    sh """
                        sonar-scanner \
                        -Dsonar.projectKey=${PROJECT_KEY} \
                        -Dsonar.sources=. \
                        -Dsonar.go.coverage.reportPaths=coverage.out 
                    """
                }
            }
        }
        stage('Quality Gate') {
            steps {
                timeout(time: 1, unit: 'HOURS') {
                    waitForQualityGate abortPipeline: true
                }
            }
        }
    }

    post {
        success {
            echo 'Pipeline succeeded'
        }
        failure {
            echo 'Pipeline failed'
        }
    }
}