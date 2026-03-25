# Set up a base OCP cluster

```
oc apply -k https://github.com/redhat-na-ssa/slinky-on-openshift/deploy/ocp-demo/operator?ref=main
```

Wait for operators to deploy

```
oc get csv -A| grep -v Succeeded
```

After operators are installed, create the custom resource instances

```
oc apply -k https://github.com/redhat-na-ssa/slinky-on-openshift/deploy/ocp-demo/instance?ref=main
```
