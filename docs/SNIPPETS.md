## Snippets

Working with this codebase may have you repeating a couple of small code fragments all over the place ("mini-boilerplate" if you may).

Here are some custom snippets may be able to help you out.

> The format of the snippets are in snipmate's custom format but it should be relatively simple to translate them to vscode or other program's snippet formats. (See snipmate examples [here](https://github.com/honza/vim-snippets/tree/master/snippets))

### Golang

```snipmate
snippet iferr
	if err != nil {
		${0:return err}
	}
snippet span
	ctx, span := tracer.Start(ctx, "${0}")
	defer span.End()
snippet recerr
	span.RecordError(err)
	span.SetStatus(codes.Error, ${0:err.Error()})
```

### Protobuf

```snipmate
snippet reqres
	// ${0}
	message ${0}Request {
	}
	message ${0}Response {
	}
snippet rpc
	rpc ${0}(${0}Request) returns (${0}Response);
```

