# filestorage

see [GoDoc.](https://godoc.org/github.com/RyoNkmr/filestorage)

## Installation

```
go get github.com/RyoNkmr/filestorage
```

## Example

```go
type Hoge struct {
	Hoge int
	Fuga *Fuga
}

type Fuga struct {
	Hoge int
}

func newHoge(i int) *Hoge {
	return &Hoge{
		Hoge: i,
		Fuga: &Fuga{
			Hoge: i,
		},
	}
}

func main() {
	fs := filestorage.NewFileStorage(".")
	a := newHoge(1)

	if err := fs.Set("1", &a, nil); err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", a) // &main.Hoge{Hoge:1, Fuga:(*main.Fuga)(0xc0000a0000)}

	expired := time.Now().Add(1 * time.Second)
	if err := fs.Set("2", &a, &expired); err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", a) // &main.Hoge{Hoge:1, Fuga:(*main.Fuga)(0xc0000a0000)}

	time.Sleep(3 * time.Second)

	var h Hoge
	if err := fs.Get("1", &h); err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", h) // main.Hoge{Hoge:1, Fuga:(*main.Fuga)(0xc0000a0028)}

	var h2 Hoge
	if err := fs.Get("2", &h2); err != nil {
		fmt.Println(err) // filetray: data has already expired
	}
	fmt.Printf("%#v\n", h2) // main.Hoge{Hoge:0, Fuga:(*main.Fuga)(nil)}

	if err := fs.Delete("1"); err != nil {
		panic(err)
	}
	if err := fs.Get("1", &h); err != nil {
		fmt.Println(err) // FileStorage: no data
	}
}
```
