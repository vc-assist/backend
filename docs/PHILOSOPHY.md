## philosophy

this codebase has a certain degree of philosophy behind its design, I will document it for consistency for future maintainers.

1. code as convention - coding conventions won't be written out in some kind of document, instead it is the **expectation** that maintainers will try to make new code follow the precedents and standards set by existing code. this means:
   1. using the existing naming/coding style conventions
   2. using an existing technologies instead of adding another one that does roughly the same thing (technologies can mean: libraries, frameworks all the way to configuration file formats)
   3. if a technology can be used in multiple ways, follow the way that it is already being used in
2. minimize magic - "magic" in software is a messy, complex thing that you can't understand or don't want to understand (understand, meaning, knowing how it works, not just how to use it). therefore it should be in the maintainer's interest to minimize the amount of magic **you** create in your codebase. what this means is:
   1. you shouldn't have magical build/setup processes (if you need to, limit the amount of interaction you have to have with them)
   2. you shouldn't have magical global packages/variables
   3. you shouldn't have magical wrappers or "conventions" around existing tools/frameworks. *use existing/standard tools as much as possible*
   4. if you must have magic, it should be separated out of this codebase and given it's own repo
3. minimize comments - you should not need to rely on comments to understand code (in most cases), if your code cannot be understood without comments it might be in need of a refactoring.

