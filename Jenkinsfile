#!/usr/bin/env groovy
properties(
    [
        buildDiscarder(
            logRotator(
                numToKeepStr: '5'
            )
        )
    ]
)

def SSH_CREDENTIAL_ID = '18270936-0906-4c40-a90e-bcf6661f501d'
def DOCKER_REGISTRY = 'quay.fnox.se'
def DOCKER_CREDENTIAL_ID = 'quay-fnox-se'

node('go1.25') {
    container('run') {
        def tag = ''
        def strippedTag = ''

        try {
            stage('Checkout') {
                checkout scm
                tag = sh(script: 'git tag -l --contains HEAD', returnStdout: true).trim()
                echo "Detected tag: ${tag ?: 'none'}"
                echo "Branch: ${env.BRANCH_NAME}"
            }

            stage('Fetch dependencies') {
                sshagent(credentials: [SSH_CREDENTIAL_ID]) {
                    sh 'go mod download'
                }
            }

            stage('Test') {
                sh 'make test'
            }

            if (env.BRANCH_NAME == 'master' && tag != '') {
                stage('Build') {
                    sh 'make build'
                }

                stage('Docker Build & Push') {
                    docker.withRegistry("https://${DOCKER_REGISTRY}", DOCKER_CREDENTIAL_ID) {
                        strippedTag = tag.replaceFirst('v', '')
                        echo "Building and pushing Docker image with tag: ${strippedTag}"
                        sh("make push VERSION=${strippedTag}")
                    }
                }

                echo "Successfully built and pushed version ${strippedTag}"
            } else {
                echo "Skipping build and push (Branch: ${env.BRANCH_NAME}, Tag: ${tag ?: 'none'})"
            }

            currentBuild.result = 'SUCCESS'
        } catch (err) {
            currentBuild.result = 'FAILED'
            echo "Build failed with error: ${err.message}"
            throw err
        }
    }
}
