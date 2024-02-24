## Description
This project defines a custom resource (`PodInfo`) that will track the desired state of a deployment of [PodInfo](https://github.com/stefanprodan/podinfo/tree/master/cmd/podinfo). Additionally, it contains a kubernetes operator will ensure the state of the cluster is in sync with the `PodInfo` custom resource.

## Getting Started

### Prerequisites
- go version v1.21.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

### To Deploy locally on a cluster (ie: MiniKube)
- Install the CRDs into the cluster: `make install`
- Run the operator locally: `make run`
- Create a CR in the cluster:
    - One is defined in `config/sample`: `kubectl apply -k config/samples/`
- Once applying the CR in the cluster, observe the `PodInfo` and `Redis` resources created in the clsuter
- Modify the CR in the cluster: `kubectl edit PodInfo <CR_NAME>`
- Observe the deployment for `PodInfo` roll the pods and roll out the changes

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Note** - Running this step should delete all resources managed by the controller

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## How to use the application
1. After a custom resource exists in the cluster, port-forward the PodInfo deployment. If using the custom resource defined in config/samples: `kubectl port-forward deployment.apps/podinfo-sample-pod-info-deployment 9898:9898`
2. Navigate to a browser and open the application: `localhost:9898`
3. Observe the color / message displayed.
4. Modify the `PodInfo` custom resource in the cluster: `kubectl edit PodInfo <CUSTOM_RESOURCE_NAME>` and change any value you want. Some common values to change are: 
    - `spec.ui.color` to change the color
    - `spec.ui.message` to change the message
    - `spec.redis.enabled` to enable /disable redis
    - `spec.replicaCount` to increase / decrease the number of replicas
5. Watch the operator's deployment logs. If using the custom resource defined in config/samples: `kubectl logs deployment.apps/podinfo-sample-pod-info-deployment`
6. Validate your changes by repeating step 2. If you are using port-forwarding, when the pods roll, you will need to re-port-forward the deployment.
7. If `spec.redis.enabled=true`, you can add a new key by making a `POST` / `PUT` request to `localhost:9898/cache/<key>`. Ex: 
```
curl --location --request PUT 'localhost:9898/cache/angi' \
--header 'Content-Type: text/plain' \
--data 'Ryan Is Awesome :)'
```
8. If `spec.redis.enabled=true`, you can retrieve the value of a given key by making a `GET` request to `localhost:9898/cache/<key>`. Ex: 
```
curl --location 'localhost:9898/cache/angi'
```


## Custom Resource Definition
This project contains a custom resource definition which is used as the source of truth for the PodInfo deployment in the cluster. The custom resource works as follows:

```
spec:
  replicaCount: <the number of pods to deploy podinfo to>
  resources:
    memoryLimit: <the memory limit allowed for the podinfo container>
    cpuRequest: <the cpu limit allowed for the podinfo container>
  image:
    repository: <the repo to pull the image from>
    tag: <the image tag>
  ui:
    color: <a hex color code to use as a background for podinfo>
    message: <a message to display in podinfo>
    cache: <the redis cache to point podinfo to>
  redis:
    enabled: <if cache should be enabled>
```

## Controllers
This project contains a controller to watch for `PodInfo` custom resources in the cluster. The reconcile loop works as follows:

1. Pull the `PodInfo` custom resource from the cluster that the loop was kicked off from.
2. If the `PodInfo` custom resource does not have redis enabled, delete the redis resources. This is used to allow for the cleanup of redis resources if redis was disabled (via editing the `PodInfo` custom resource).
3. Check if the `PodInfo` custom resource has been deleted, if not, add a finalizer to the `PodInfo` custom resource, if it does not already exist.
4. If the `PodInfo` custom resource has been deleted, clean up the PodInfo resources and the redis resources in the cluster. Once that is done, remove the finalizer so the `PodInfo` custom resource can be deleted by the api server.

## Possible Enhancements

- Add monitoring for the operator
- Add auto-scaling and pod disruption budgets to the various deployments
- Move the creation of in cluster resources (deployments / services / configmaps / etc) into Helm
- Expose the PodInfo deployment with a LoadBalancer service

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

