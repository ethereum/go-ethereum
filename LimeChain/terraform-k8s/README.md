This is terraform code that will create a kubernetes cluster create deployment there. Please perform the following steps.

1. Install latest `aws cli`
2. Export credential

```
export AWS_ACCESS_KEY_ID=<>
export AWS_SECRET_ACCESS_KEY=<>
export AWS_SESSION_TOKEN=<>
```

3. Run `terraform init`, `terrafrom apply`
4. Set up kubernetes credentials via `aws eks --region $(terraform output -raw region) update-kubeconfig --name $(terraform output -raw cluster_name)`
