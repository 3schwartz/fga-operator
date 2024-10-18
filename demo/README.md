# References
- https://github.com/3schwartz/fga-operator
- https://3schwartz.github.io/fga-operator/

# Done before demo
Change secret in `./development/helm/templates/secrets.yaml` to
```sh
---
kind: Secret
apiVersion: v1
metadata:
  name: openfga
stringData:
  uri: postgres://postgres:password@openfga-postgresql.demo.svc.cluster.local:5432/openfga?sslmode=disable
```

Run below
```sh
k create namespace demo

cd ./development/helm
helm upgrade --install openfga .

helm repo add fga-operator https://3schwartz.github.io/fga-operator/
helm repo update
```

# Done at demo

```sh
helm search repo fga --devel
```

```sh
helm install fga-operator fga-operator/fga-operator --version 0.1.0-<TAG> -f values.yaml
```

```sh
k apply -f authorization_model_v1.yaml
```

```sh
k apply -f example-deployment.yaml
```

```sh
k apply -f authorization_model_v2.yaml
```

// Change version
```sh
k apply -f example-deployment.yaml 
```
