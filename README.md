# Chaos Tank

The most fun you'll have obliterating your production cluster!

## DISCLAIMER

Whatever you shoot are the pods in the `default` namespace. Please make sure you are not actually killing your environments!

## Usage

```
go get
go run chaos_tank.go -kubeconfig ~/.kube/config 
```

## Example

![Alt text](target_destroyed.png?raw=true "Target destroyed")
