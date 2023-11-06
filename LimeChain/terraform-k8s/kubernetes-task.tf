provider "kubernetes" {
  host                   = module.eks.cluster_endpoint
  cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    args        = ["eks", "get-token", "--cluster-name", local.cluster_name]
    command     = "aws"
  }
}




resource "kubernetes_deployment" "ethereum-node" {
  metadata {
    name = "ethereum-node"
    labels = {
      app = "eth-dev"
    }
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        app = "eth-dev"
      }
    }

    template {
      metadata {
        labels = {
          app = "eth-dev"
        }
      }

      spec {
        container {
          image = "vatman/client-go-hardhat:latest"
          name  = "ethereum-node"
          args  = ["--dev", "--http", "--http.api", "eth,web3,net", "--http.corsdomain", "http://remix.ethereum.org"]
        }
      }
    }
  }
}



resource "kubernetes_service" "example" {
  metadata {
    name = "ethereum-node-service"
  }
  spec {
    selector = {
      app = "eth-dev"
    }

    port {
      port        = 8545
      target_port = 8545
    }

    type = "ClusterIP"
  }
}