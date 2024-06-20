## Technical Justifications

> Note: Points of justification are not in any particular order (ie. #1 is not necessarily more significant than #3).

### Why Go?

1. Simplicity: Go has simple syntax, language constructs are easy to understand, and it doesn't come with many language features (which is a good thing, you don't need to learn like 30 language constructs to read someone else's code, compare this with C#, Java, Swift or Rust).
2. Batteries included: Go's standard library is very powerful, it can do many things that you'd need 3rd party libraries to do in other languages which also allows for a cleaner architecture in general, as many constructs are standardized.
3. Error handling: Go does errors as values (other languages do as well), but this forces you to think about error handling instead of getting surprised when things fail in production, or preemptively slathering try/catch everywhere.
4. Performance: While Go isn't the fastest language (leave that up to C++/Rust/C/Zig/other low level managed memory things), it is way faster than your JIT/interpreted languages without adding a lot of work in terms of thinking about memory.
5. Memory: Go also uses way less memory than JIT/interpreted languages because it's statically typed and compiled. This means we can get by with a $5 AWS lightsail instance without having it crash due to OOM errors every week.
6. Tooling: Go has probably the best tooling of any language out there. The compiler is extremely fast, the package manager, formatter, and LSP are all fast, efficient and reliable. It outputs a single binary, no need for docker images & build scripts. In general, everything just works, and it works well, no need to spend engineering hours debugging your tools/build chain. (this is in great contrast to languages like JavaScript, C or C++)

### Why JSON5 as a configuration format, and not yaml/toml/regular json/etc...?

1. It's much nicer to write than plain json (trailing commas, comments, more number formats, etc...)
2. It's more strict than yaml (no guessing what is or is not a string, indentation not significant, very explicit about what's a list, what's an object, etc...)
3. It's simple, basically JSON with syntax sugar, doesn't have as many features as TOML or YAML
4. Libraries to parse it are widely found and don't usually have bugs, only drawback is that LSP support is not widespread.

### Why gRPC+protobufs instead of OpenAPI, REST or graphQL?

1. Type safety, so no REST, this is especially important when you have a lot of different types, and types nested within them. it is good to have a single source of truth and a strict contract.
2. The only real benefit OpenAPI brings over gRPC is that it's human readable if you're doing interception with wireshark. this isn't important in our case since we'll probably never need that.
3. GraphQL comes with a bunch of added complexity for virtually no benefit (in our case), it's better to choose a simpler format.

### Why sqlc instead of <any other go ORM or raw sql>?

1. Simplicity, you literally write raw SQL then generate some boilerplate to use it. This comes with many benefits in terms of performance and the fact that you don't need to learn some kind of ORM abstraction, all you need to know is SQL.
2. Type safety, sqlc comes with linters/parsers to validate your SQL and make sure it's type 
safe, generated boilerplate is also ensured to be type safe.
3. Other tools can also be used in tandem with sqlc to produce migrations/etc...

