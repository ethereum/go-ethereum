provider "kubernetes" {
  config_path = "~/.kube/config"
}

resource "kubernetes_namespace" "example" {
  metadata {
    name = "example"
  }
}

resource "kubernetes_deployment" "go-ethereum" {
  metadata {
    name      = "go-ethereum"
    namespace = kubernetes_namespace.example.metadata[0].name
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        app = "go-ethereum"
      }
    }

    template {
      metadata {
        labels = {
          app = "go-ethereum"
        }
      }

      spec {
        container {
          image = "limetester/go-ethereum:latest"
          name  = "go-ethereum"

          ports {
            container_port = 8545
          }
        }
      }
    }
  }
}

