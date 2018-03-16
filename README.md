# simplesync

```
func main() {
  workers := simplesync.NewWorkerPool(8)
  workers.Execute(func(i int) {
    fmt.Printf("Hello World! %v\n", i)
  })
}
```
