# opentelemetry-mackerel playgrounds
Sample code to send OpenTelemetry metrics and traces to Mackerel (Vaxila)

## prerequisites
- Create a Artifact Registry image repository (`opentelemetry-mackerel-playgrounds`)
  - https://cloud.google.com/artifact-registry/docs/docker/store-docker-container-images?hl=ja
- Create secret (`mackerel_apikey`) via Secret Manager
  - https://mackerel.io/ja/api-docs/
  - https://cloud.google.com/secret-manager/docs/creating-and-accessing-secrets?hl=ja
- Setup Workload Identity Federation (To deploy to Cloud run via GitHub Actions)
  - https://cloud.google.com/blog/ja/products/devops-sre/deploy-to-cloud-run-with-github-actions/

## send HTTP request

```console
SERVICE_URL=$(gcloud run services describe --region asia-northeast1  opentelemetry-mackerel-playgrounds --format=json | jq -r '.status.url')
curl -H "Authorization: Bearer $(gcloud auth print-identity-token)" $SERVICE_URL/hello
```