# OpenFGA controller


Help deploy autohorization model and have deployments use latest deployed authorization model.

# Steps

## Make a model


Make a authorization request 

```
apiVersion: extensions.openfga-controller/v1
kind: AuthorizationModelRequest
metadata:
  labels:
    app.kubernetes.io/name: operator
    app.kubernetes.io/managed-by: kustomize
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

It will create a store

```
apiVersion: extensions.openfga-controller/v1
kind: Store
metadata:
  creationTimestamp: "2024-06-27T08:48:10Z"
  generation: 1
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
    uid: cd01de00-c46c-468b-9ae7-b3d03d4d4b81
  resourceVersion: "579073"
  uid: 3d7b6e6b-379a-4178-9e32-73d2207bdd7b
spec:
  id: 01J1CE2YAH98MKN2SZ8BJ0XYPZ
```

and a authorization model

```
apiVersion: extensions.openfga-controller/v1
kind: AuthorizationModel
metadata:
  creationTimestamp: "2024-05-26T13:04:07Z"
  generation: 4
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
    uid: cd01de00-c46c-468b-9ae7-b3d03d4d4b81
  resourceVersion: "300835"
  uid: f61d4696-82b9-4be0-a81f-e995126d5176
spec:
  authorizationModel: "model\n  schema 1.1\n  \ntype user\n  \ntype document\n  relations\n
    \   define foo: [user]\n    define reader: [user]\n    define writer: [user]\n
    \   define owner: [user]\n"
  instance:
    createdAt: "2024-05-26T13:11:59Z"
    id: 01HYTGF0VHRM5ASHSBJJRQG87N
    version: 1.1.1
  latestModels:
  - createdAt: "2024-05-26T13:04:07Z"
    id: 01HYTG0KP32RBT84W21XYBPSGH
  - createdAt: "2024-05-26T13:09:56Z"
    id: 01HYTGB5J1NJNKQZGEYKYZQYXP
    version: 1.1.0
  - createdAt: "2024-05-26T13:10:24Z"
    id: 01HYTGC48CPR3C6412KF7T1AVX
    version: 1.1.1
```


Given a deployment with below labels, where version isn't set, ´OPENFGA_AUTH_MODEL_ID´ will be set to latest created authorization model
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

It will get environment variables updated as below
```
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "7"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"labels":{"app":"annotated-curl","openfga-store":"documents"},"name":"annotated-curl","namespace":"default"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"annotated-curl"}},"template":{"metadata":{"labels":{"app":"annotated-curl"}},"spec":{"containers":[{"command":["sleep","9999999"],"env":[{"name":"foo","value":"bar"}],"image":"curlimages/curl:8.7.1","name":"main"}]}}}}
    openfga-auth-id-updated-at: "2024-05-26T13:11:59Z"
    openfga-auth-model-version: 1.1.1
    openfga-store-id-updated-at: "2024-06-27T08:48:03Z"
  creationTimestamp: "2024-05-26T13:01:28Z"
  generation: 7
  labels:
    app: annotated-curl
    openfga-store: documents
  name: annotated-curl
  namespace: default
  resourceVersion: "579103"
  uid: f3c4241d-0d1d-4d34-ba63-80649b571ad6
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: annotated-curl
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: annotated-curl
    spec:
      containers:
      - command:
        - sleep
        - "9999999"
        env:
        - name: foo
          value: bar
        - name: OPENFGA_STORE_ID
          value: 01J1CE2YAH98MKN2SZ8BJ0XYPZ
        - name: OPENFGA_AUTH_MODEL_ID
          value: 01HYTGF0VHRM5ASHSBJJRQG87N
        image: curlimages/curl:8.7.1
        imagePullPolicy: IfNotPresent
        name: main
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: 1
  conditions:
  - lastTransitionTime: "2024-06-03T06:35:55Z"
    lastUpdateTime: "2024-06-03T06:35:55Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: "2024-05-26T13:01:28Z"
    lastUpdateTime: "2024-06-27T08:48:20Z"
    message: ReplicaSet "annotated-curl-56cb765975" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 7
  readyReplicas: 1
  replicas: 1
  updatedReplicas: 1
```

## Updating model

Now update the authorization model by making a request like below. The important part is change of the `authorizationModel` since this is the part which is compared.

```
apiVersion: extensions.openfga-controller/v1
kind: AuthorizationModelRequest
metadata:
  labels:
    app.kubernetes.io/name: operator
    app.kubernetes.io/managed-by: kustomize
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

The authorization model will then update to
```
apiVersion: extensions.openfga-controller/v1
kind: AuthorizationModel
metadata:
  creationTimestamp: "2024-05-26T13:04:07Z"
  generation: 5
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
    uid: cd01de00-c46c-468b-9ae7-b3d03d4d4b81
  resourceVersion: "579686"
  uid: f61d4696-82b9-4be0-a81f-e995126d5176
spec:
  authorizationModel: "model\n  schema 1.1\n  \ntype user\n  \ntype document\n  relations\n
    \   define foo: [user]\n    define bar: [user]\n    define reader: [user]\n    define
    writer: [user]\n    define owner: [user]\n"
  instance:
    createdAt: "2024-06-27T09:14:27Z"
    id: 01J1CFK2XZMFYW3QP1GKH2R4KN
    version: 1.1.2
  latestModels:
  - createdAt: "2024-05-26T13:04:07Z"
    id: 01HYTG0KP32RBT84W21XYBPSGH
  - createdAt: "2024-05-26T13:09:56Z"
    id: 01HYTGB5J1NJNKQZGEYKYZQYXP
    version: 1.1.0
  - createdAt: "2024-05-26T13:10:24Z"
    id: 01HYTGC48CPR3C6412KF7T1AVX
    version: 1.1.1
  - createdAt: "2024-05-26T13:11:59Z"
    id: 01HYTGF0VHRM5ASHSBJJRQG87N
    version: 1.1.1
```

and the annotated deployment will have `OPENFGA_AUTH_MODEL_ID` updated.
```
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "8"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"labels":{"app":"annotated-curl","openfga-store":"documents"},"name":"annotated-curl","namespace":"default"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"annotated-curl"}},"template":{"metadata":{"labels":{"app":"annotated-curl"}},"spec":{"containers":[{"command":["sleep","9999999"],"env":[{"name":"foo","value":"bar"}],"image":"curlimages/curl:8.7.1","name":"main"}]}}}}
    openfga-auth-id-updated-at: "2024-06-27T09:14:27Z"
    openfga-auth-model-version: 1.1.2
    openfga-store-id-updated-at: "2024-06-27T08:48:03Z"
  creationTimestamp: "2024-05-26T13:01:28Z"
  generation: 8
  labels:
    app: annotated-curl
    openfga-store: documents
  name: annotated-curl
  namespace: default
  resourceVersion: "579714"
  uid: f3c4241d-0d1d-4d34-ba63-80649b571ad6
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: annotated-curl
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: annotated-curl
    spec:
      containers:
      - command:
        - sleep
        - "9999999"
        env:
        - name: foo
          value: bar
        - name: OPENFGA_STORE_ID
          value: 01J1CE2YAH98MKN2SZ8BJ0XYPZ
        - name: OPENFGA_AUTH_MODEL_ID
          value: 01J1CFK2XZMFYW3QP1GKH2R4KN
        image: curlimages/curl:8.7.1
        imagePullPolicy: IfNotPresent
        name: main
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: 1
  conditions:
  - lastTransitionTime: "2024-06-03T06:35:55Z"
    lastUpdateTime: "2024-06-03T06:35:55Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: "2024-05-26T13:01:28Z"
    lastUpdateTime: "2024-06-27T09:14:29Z"
    message: ReplicaSet "annotated-curl-6c6496b759" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 8
  readyReplicas: 1
  replicas: 1
  updatedReplicas: 1
```

## Set specific version on deployment

Let's say you want to lock the `OPENFGA_AUTH_MODEL_ID` to a specific version. The add label `openfga-auth-model-version`.

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

By applying above we see that `OPENFGA_AUTH_MODEL_ID` is changed to the authorization model with version label `1.1.1`.

```
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "9"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"labels":{"app":"annotated-curl","openfga-auth-model-version":"1.1.1","openfga-store":"documents"},"name":"annotated-curl","namespace":"default"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"annotated-curl"}},"template":{"metadata":{"labels":{"app":"annotated-curl"}},"spec":{"containers":[{"command":["sleep","9999999"],"env":[{"name":"foo","value":"bar"}],"image":"curlimages/curl:8.7.1","name":"main"}]}}}}
    openfga-auth-id-updated-at: "2024-06-27T09:20:14Z"
    openfga-auth-model-version: 1.1.1
    openfga-store-id-updated-at: "2024-06-27T08:48:03Z"
  creationTimestamp: "2024-05-26T13:01:28Z"
  generation: 10
  labels:
    app: annotated-curl
    openfga-auth-model-version: 1.1.1
    openfga-store: documents
  name: annotated-curl
  namespace: default
  resourceVersion: "579851"
  uid: f3c4241d-0d1d-4d34-ba63-80649b571ad6
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: annotated-curl
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: annotated-curl
    spec:
      containers:
      - command:
        - sleep
        - "9999999"
        env:
        - name: foo
          value: bar
        - name: OPENFGA_STORE_ID
          value: 01J1CE2YAH98MKN2SZ8BJ0XYPZ
        - name: OPENFGA_AUTH_MODEL_ID
          value: 01HYTGF0VHRM5ASHSBJJRQG87N
        image: curlimages/curl:8.7.1
        imagePullPolicy: IfNotPresent
        name: main
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: 1
  conditions:
  - lastTransitionTime: "2024-06-03T06:35:55Z"
    lastUpdateTime: "2024-06-03T06:35:55Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: "2024-05-26T13:01:28Z"
    lastUpdateTime: "2024-06-27T09:20:16Z"
    message: ReplicaSet "annotated-curl-56cb765975" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 10
  readyReplicas: 1
  replicas: 1
  updatedReplicas: 1
```
