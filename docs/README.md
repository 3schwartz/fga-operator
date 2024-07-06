# FGA Operator Documentation

This documentation helps you deploy an OpenFGA authorization model and ensures your deployments stay in sync with the latest authorization model from OpenFGA. The FGA operator automates the synchronization between your deployments and the authorization models.

## Steps

### 1. Create an Authorization Model

Make an authorization request:

```yaml
apiVersion: extensions.fga-operator/v1
kind: AuthorizationModelRequest
metadata:
  name: documents
spec:
  instances:
    - version:
        major: 1
        minor: 1
        patch: 1
      authorizationModel: |
        model
          schema 1.1
          
        type user
          
        type document
          relations
            define reader: [user]
            define writer: [user]
            define owner: [user]
```

This request will:
- Create a store with the same name in OpenFGA.
- Create a Kubernetes resource `Store`.

```yaml
apiVersion: extensions.fga-operator/v1
kind: Store
metadata:
  labels:
    authorization-model: documents
  name: documents
  namespace: default
  ownerReferences:
  - apiVersion: extensions.fga-operator/v1
    blockOwnerDeletion: true
    controller: true
    kind: AuthorizationModelRequest
    name: documents
    uid: <SOME_ID>
spec:
  id: 01J1N8HCY7MQP4QP3GVDWTM9ZG

```

- Create the authorization model in OpenFGA.
- Save the authorization model ID in a Kubernetes resource `AuthorizationModel`.

```yaml
apiVersion: extensions.fga-operator/v1
kind: AuthorizationModel
metadata:
  labels:
    authorization-model: documents
  name: documents
  namespace: default
  ownerReferences:
  - apiVersion: extensions.fga-operator/v1
    blockOwnerDeletion: true
    controller: true
    kind: AuthorizationModelRequest
    name: documents
    uid: <SOME_ID>
spec:
  instances:
  - authorizationModel: |
      model
        schema 1.1

      type user

      type document
        relations
          define reader: [user]
          define writer: [user]
          define owner: [user]  
    createdAt: "2024-07-06T06:44:24Z"
    id: 01J23CJTA8X4K87X62ECX1Y58Z
    version:
      major: 1
      minor: 1
      patch: 1
```

### 2. Deployment with Label

Given a deployment with the label `openfga-store` set to the name of the authorization request:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    fga-store: documents
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
    openfga-auth-id-updated-at: "2024-07-06T06:44:24Z"
    openfga-auth-model-version: 1.1.1
    openfga-store-id-updated-at: "2024-07-06T06:44:24Z"
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
          value: 01J1N8HCY7MQP4QP3GVDWTM9ZG
        - name: OPENFGA_AUTH_MODEL_ID
          value: 01J23CJTA8X4K87X62ECX1Y58Z
        image: curlimages/curl:8.7.1
        imagePullPolicy: IfNotPresent
        name: main
````

### 3. Update the Authorization Model
To update the authorization model, make a request like below. The important part is setting a new version since this is what the controller compares.

```yaml
apiVersion: extensions.fga-operator/v1
kind: AuthorizationModelRequest
metadata:
  name: documents
spec:
  instances:
    - version:
        major: 1
        minor: 1
        patch: 2
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
    - version:
        major: 1
        minor: 1
        patch: 1
      authorizationModel: |
        model
          schema 1.1
          
        type user
          
        type document
          relations
            define reader: [user]
            define writer: [user]
            define owner: [user]
```

The controller will call OpenFGA and create the new authorization model. The controller will update the AuthorizationModel with the new reference.

```yaml
apiVersion: extensions.fga-operator/v1
kind: AuthorizationModel
metadata:
  creationTimestamp: "2024-07-06T06:44:25Z"
  generation: 2
  labels:
    authorization-model: documents
  name: documents
  namespace: default
  ownerReferences:
  - apiVersion: extensions.fga-operator/v1
    blockOwnerDeletion: true
    controller: true
    kind: AuthorizationModelRequest
    name: documents
    uid: e507bff2-09b2-44d7-9f2e-b6f238dda3b3
  resourceVersion: "734046"
  uid: 6c0ab77c-765b-40d1-8372-77a5cefb325e
spec:
  instances:
  - authorizationModel: |
      model
        schema 1.1

      type user

      type document
        relations
          define reader: [user]
          define writer: [user]
          define owner: [user]
    createdAt: "2024-07-06T06:44:24Z"
    id: 01J23CJTA8X4K87X62ECX1Y58Z
    version:
      major: 1
      minor: 1
      patch: 1
  - authorizationModel: |
      model
        schema 1.1

      type user

      type document
        relations
          define foo: [user]
          define reader: [user]
          define writer: [user]
          define owner: [user]
    createdAt: "2024-07-06T06:50:37Z"
    id: 01J23CY66445FA3Z6TQEC9WBZK
    version:
      major: 1
      minor: 1
      patch: 2
```

The controller will update annotated deployments so that the example deployment will have its `OPENFGA_AUTH_MODEL_ID` environment variable updated.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    openfga-auth-id-updated-at: "2024-07-06T06:50:37Z"
    openfga-auth-model-version: 1.1.2
    openfga-store-id-updated-at: "2024-07-06T06:44:24Z"
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
          value: 01J1N8HCY7MQP4QP3GVDWTM9ZG
        - name: OPENFGA_AUTH_MODEL_ID
          value: 01J23CY66445FA3Z6TQEC9WBZK
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
    openfga-auth-id-updated-at: "2024-07-06T07:01:40Z"
    openfga-auth-model-version: 1.1.1
    openfga-store-id-updated-at: "2024-07-06T06:44:24Z"
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
          value: 01J1N8HCY7MQP4QP3GVDWTM9ZG
        - name: OPENFGA_AUTH_MODEL_ID
          value: 01J23CJTA8X4K87X62ECX1Y58Z
        image: curlimages/curl:8.7.1
        imagePullPolicy: IfNotPresent
        name: main
```