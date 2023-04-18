credentialDocker = 'dockerhub'

pipeline {
    agent any
    options {
        timeout (20)
    }
    tools {
        go 'go-1.18'
        nodejs "nodejs"
    }
    environment {
        GO111MODULE = 'on'
        PATH="/home/ubuntu/.cargo/bin:$PATH"
        // LOG_DOCKER = 'true'
    }
    stages {
        stage('Tag') {
            steps {
                script {
                    TAGNAME = sh(returnStdout: true, script: 'git tag -l --points-at HEAD')
                    sh "echo ${TAGNAME}"
                    // ... 
                }
            }
        }
        stage('Build') {
            environment {
                DOCKER_CREDENTIALS = credentials('dockerhub')
            }
           steps {         
                withCredentials([usernamePassword(credentialsId: "${credentialDocker}", passwordVariable: 'dockerPassword', usernameVariable: 'dockerUser')]) {
                        // Use a scripted pipeline.
                        script {
                            stage('Push image') { 
                                    if (TAGNAME == ""){
                                        return;
                                    }
                                    sh "docker login --username=${dockerUser} --password=${dockerPassword}"
                                    sh "docker build -t scrolltech/l2geth:latest ."
                                    sh "docker tag scrolltech/l2geth:latest scrolltech/l2geth:${TAGNAME}"
                                    sh "docker push scrolltech/l2geth:${TAGNAME}"                
                                }
                        }
                    }
                }
            }
    }
    post {
          success {
            slackSend(message: "l2geth tag ${TAGNAME} build dockersSuccessed")
          }
          // triggered when red sign
          failure {
            slackSend(message: "l2geth tag ${TAGNAME} build docker failed")
          }
          always {
            cleanWs() 
        }
    }
}
