pipeline {
    agent any
    tools {
        go 'go-1.17'
    }
    environment {
        GO111MODULE = 'on'
    }
    stages {
        stage('Build') {
            steps {
                // Get some code from a GitHub repository
                 
                // git branch: 'zkrollup',
                //     credentialsId: 'testgitchuhan1',
                //     url: 'git@github.com:scroll-tech/go-ethereum.git'
                    
                // Build the app.
                sh 'go build' 
                    
            }

        }
        stage('Test') {
            // Use golang.
            steps {                 
                // Remove cached test results.
                sh 'go clean -cache'
                // Run Unit Tests.
                sh 'make test'            
            }
        }      

        stage('Docker') {         
            environment {
                // Extract the username and password of our credentials into "DOCKER_CREDENTIALS_USR" and "DOCKER_CREDENTIALS_PSW".
                // (NOTE 1: DOCKER_CREDENTIALS will be set to "your_username:your_password".)
                // The new variables will always be YOUR_VARIABLE_NAME + _USR and _PSW.
                // (NOTE 2: You can't print credentials in the pipeline for security reasons.)
                DOCKER_CREDENTIALS = credentials('dockerhub')
            }

            steps {                           
                // Use a scripted pipeline.
                script {
                    node {
                        def app
                        //  stage('Initialize') {
                        //     def dockerHome = tool 'myDocker'
                        //     env.PATH = "${dockerHome}/bin:${env.PATH}"
                        // }

                        stage('Build image') {
                            app = docker.build("${env.DOCKER_CREDENTIALS_USR}/l2geth-img")
                        }

                        stage('Push image') {  
                            // Use the Credential ID of the Docker Hub Credentials we added to Jenkins.
                            docker.withRegistry('https://registry.hub.docker.com', 'dockerhub') {                                
                                // Push image and tag it with our build number for versioning purposes.
                                app.push("${env.BUILD_NUMBER}")                      

                                // Push the same image and tag it as the latest version (appears at the top of our version list).
                                app.push("latest")
                            }
                        }              
                    }                 
                }
            }
        }
    }
}