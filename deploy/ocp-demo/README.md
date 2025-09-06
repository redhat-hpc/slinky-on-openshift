# Set up a base OCP cluster

```
oc apply -k https://github.com/redhat-na-ssa/slinky-on-openshift/deploy/ocp-demo/operator?ref=main
```

Once operators have installed (30s?)

```
oc apply -k https://github.com/redhat-na-ssa/slinky-on-openshift/deploy/ocp-demo/instance?ref=main
```
