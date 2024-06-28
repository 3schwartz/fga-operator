# OpenFGA Controller Documentation

This documentation helps you deploy an authorization model and ensures your deployments stay in sync with the latest authorization model from OpenFGA. The OpenFGA controller automates the synchronization between your deployments and the authorization models.

## Steps

### 1. Create an Authorization Model

Make an authorization request:

```yaml
apiVersion: extensions.openfga-controller/v1
kind: AuthorizationModelRequest
metadata:
  name: documents
spec:
  version: 1.1.1
  authorizationModel: |
    model
      schema 1.1
      
    type user
      
    type document
      relations
        define foo: [user]
        define reader: [user]
        define writer: [user]
        define owner: [user]
```

This request will:
- Create a store with the same name in OpenFGA.
- Create a Kubernetes resource `Store`.

```yaml
apiVersion: extensions.openfga-controller/v1
kind: Store
metadata:
  labels:
    authorization-model: documents
  name: documents
  namespace: default
  ownerReferences:
  - apiVersion: extensions.openfga-controller/v1
    blockOwnerDeletion: true
    controller: true
    kind: AuthorizationModelRequest
    name: documents
    uid: <SOME_ID>
spec:
  id: 01J1CE2YAH98MKN2SZ8BJ0XYPZ

```

- Create the authorization model in OpenFGA.
- Save the authorization model ID in a Kubernetes resource `AuthorizationModel`.

```yaml
apiVersion: extensions.openfga-controller/v1
kind: AuthorizationModel
metadata:
  labels:
    authorization-model: documents
  name: documents
  namespace: default
  ownerReferences:
  - apiVersion: extensions.openfga-controller/v1
    blockOwnerDeletion: true
    controller: true
    kind: AuthorizationModelRequest
    name: documents
    uid: <SOME_ID>
spec:
  authorizationModel: "model\n  schema 1.1\n  \ntype user\n  \ntype document\n  relations\n
    \   define foo: [user]\n    define reader: [user]\n    define writer: [user]\n
    \   define owner: [user]\n"
  instance:
    createdAt: "2024-05-26T13:11:59Z"
    id: 01HYTGF0VHRM5ASHSBJJRQG87N
    version: 1.1.1
  latestModels: []
```

### 2. Deployment with Label

Given a deployment with the label `openfga-store` set to the name of the authorization request:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    openfga-store: documents
    app: annotated-curl
  name: annotated-curl
spec:
  replicas: 1
  selector:
    matchLabels:
      app: annotated-curl
  template:
    metadata:
      labels:
        app: annotated-curl
    spec:
      containers:
      - name: main
        image: curlimages/curl:8.7.1
        command: ["sleep", "9999999"]
```

The environment variable `OPENFGA_AUTH_MODEL_ID` will be set to the latest created authorization model ID from OpenFGA.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    openfga-auth-id-updated-at: "2024-05-26T13:11:59Z"
    openfga-auth-model-version: 1.1.1
    openfga-store-id-updated-at: "2024-06-27T08:48:03Z"
  labels:
    app: annotated-curl
    openfga-store: documents
  name: annotated-curl
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: annotated-curl
  template:
    metadata:
      labels:
        app: annotated-curl
    spec:
      containers:
      - command:
        - sleep
        - "9999999"
        env:
        - name: OPENFGA_STORE_ID
          value: 01J1CE2YAH98MKN2SZ8BJ0XYPZ
        - name: OPENFGA_AUTH_MODEL_ID
          value: 01HYTGF0VHRM5ASHSBJJRQG87N
        image: curlimages/curl:8.7.1
        imagePullPolicy: IfNotPresent
        name: main
````

### 3. Update the Authorization Model
To update the authorization model, make a request like below. The important part is the change in the authorizationModel since this is what the controller compares.

```yaml
apiVersion: extensions.openfga-controller/v1
kind: AuthorizationModelRequest
metadata:
  name: documents
spec:
  version: 1.1.2
  authorizationModel: |
    model
      schema 1.1
      
    type user
      
    type document
      relations
        define foo: [user]
        define bar: [user]
        define reader: [user]
        define writer: [user]
        define owner: [user]
```

The controller will call OpenFGA and create the new authorization model. The controller will update the AuthorizationModel with the new reference and move the old instance to `latestModels`.

```yaml
apiVersion: extensions.openfga-controller/v1
kind: AuthorizationModel
metadata:
  labels:
    authorization-model: documents
  name: documents
  namespace: default
  ownerReferences:
  - apiVersion: extensions.openfga-controller/v1
    blockOwnerDeletion: true
    controller: true
    kind: AuthorizationModelRequest
    name: documents
    uid: <SOME_ID>
spec:
  authorizationModel: "model\n  schema 1.1\n  \ntype user\n  \ntype document\n  relations\n
    \   define foo: [user]\n    define bar: [user]\n    define reader: [user]\n    define
    writer: [user]\n    define owner: [user]\n"
  instance:
    createdAt: "2024-06-27T09:14:27Z"
    id: 01J1CFK2XZMFYW3QP1GKH2R4KN
    version: 1.1.2
  latestModels:
  - createdAt: "2024-05-26T13:11:59Z"
    id: 01HYTGF0VHRM5ASHSBJJRQG87N
    version: 1.1.1
```

The controller will update annotated deployments so that the example deployment will have its `OPENFGA_AUTH_MODEL_ID` environment variable updated.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    openfga-auth-id-updated-at: "2024-06-27T09:14:27Z"
    openfga-auth-model-version: 1.1.2
    openfga-store-id-updated-at: "2024-06-27T08:48:03Z"
  labels:
    app: annotated-curl
    openfga-store: documents
  name: annotated-curl
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: annotated-curl
  template:
    metadata:
      labels:
        app: annotated-curl
    spec:
      containers:
      - command:
        - sleep
        - "9999999"
        env:
        - name: OPENFGA_STORE_ID
          value: 01J1CE2YAH98MKN2SZ8BJ0XYPZ
        - name: OPENFGA_AUTH_MODEL_ID
          value: 01J1CFK2XZMFYW3QP1GKH2R4KN
        image: curlimages/curl:8.7.1
        imagePullPolicy: IfNotPresent
        name: main
```

### 4. Set a Specific Version on Deployment

To lock the `OPENFGA_AUTH_MODEL_ID` to a specific user-provided version, add the label `openfga-auth-model-version` and set it to the desired version.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    openfga-store: documents
    openfga-auth-model-version: "1.1.1"
    app: annotated-curl
  name: annotated-curl
spec:
  replicas: 1
  selector:
    matchLabels:
      app: annotated-curl
  template:
    metadata:
      labels:
        app: annotated-curl
    spec:
      containers:
      - name: main
        image: curlimages/curl:8.7.1
        command: ["sleep", "9999999"]
```

By applying the above, the `OPENFGA_AUTH_MODEL_ID` will be set to the authorization model ID with version label `1.1.1`.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    openfga-auth-id-updated-at: "2024-06-27T09:20:14Z"
    openfga-auth-model-version: 1.1.1
    openfga-store-id-updated-at: "2024-06-27T08:48:03Z"
  labels:
    app: annotated-curl
    openfga-auth-model-version: 1.1.1
    openfga-store: documents
  name: annotated-curl
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: annotated-curl
  template:
    metadata:
      labels:
        app: annotated-curl
    spec:
      containers:
      - command:
        - sleep
        - "9999999"
        env:
        - name: OPENFGA_STORE_ID
          value: 01J1CE2YAH98MKN2SZ8BJ0XYPZ
        - name: OPENFGA_AUTH_MODEL_ID
          value: 01HYTGF0VHRM5ASHSBJJRQG87N
        image: curlimages/curl:8.7.1
        imagePullPolicy: IfNotPresent
        name: main
```
