## Folder information

This folder contains the buit'n and the default markdown response engine support for the [Iris web framework](https://github.com/kataras/iris).


## Install

```sh
$ go get -u github.com/iris-contrib/response/markdown
```

Because the markdown is text/html as content-type but I don't want to override
the real simple html content you may render with `context.HTML`, this response engine
will be registered with custom key/type `text/markdown`, so:

```go
iris.UseResponse(markdown.New(), markdown.ContentType)

// the context.Markdown will look up for text/markdown keys.
// you can still use any name you want and render it with `context.Render("my custom name",...)`
```

## How to use

- Docs [here](https://kataras.gitbooks.io/iris/content/render_response.html)
- Examples [here](https://github.com/iris-contrib/examples/tree/master/response_engines)
