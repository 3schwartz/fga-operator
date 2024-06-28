# OpenFGA controller


Help deploy autohorization model and have deployments use latest deployed authorization model.

# Steps

## Make a model


Make a authorization request 

```
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

It will create a store with the same name in OpenFGA and make Kubernetes resource `Store`.
```
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

It will also create the authorization model in OpenFGA and save the authorization model id in a Kubernetes resource `AuthorizationModel` which acts
as a reference between the authroization model id in OpenFGA and the user provided name and version.
```
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

Given a deployment with label `openfga-store` with value same as name of the request like below
```
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

Then ´OPENFGA_AUTH_MODEL_ID´ environemnt variable will be set to latest created authorization model id from OpenFGA. It will get environment variables updated as below
```
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
```

## Updating model

Now update the authorization model by making a request like below. The important part is change of the `authorizationModel` since this is the part which is compared in the controller.
```
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

The controller will call OpenFGA and create the new authorization model. The controller will update the `AuthorizationModel` with the new reference.
It will move the old instance to `latestModels`.
```
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

The controller will update annotated deployments such that the example deployment will get it's `OPENFGA_AUTH_MODEL_ID` environment variable updated.
```
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
        name: main
```

## Set specific version on deployment

Let's say you want to lock the `OPENFGA_AUTH_MODEL_ID` to a specific user provided version. Then add label `openfga-auth-model-version` and set it to the version desired.

```
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

By applying above we see that `OPENFGA_AUTH_MODEL_ID` is changed to the authorization model id with version label `1.1.1`.

```
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
