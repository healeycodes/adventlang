# 🎅 Adventlang

A strongly typed but highly dynamic programming language interpreter written in Go.

```js
log("Hello, World!");
```

### Why

In the second half of November, I designed and implemented this programming language for Advent of Code (AoC). I'll be using it to solve AoC's daily puzzles and adding to the standard library as I go. Will this language make it easier for you to solve the puzzles? No, certainly not. Here be dragons, etc. But it will increase my level of fun as I tap into the joyous energy that comes with forced-creativity-through-restriction.

### Getting Started

Look in the [tests](/tests) folder for examples of how to use most language features.

If you're developing, you can use `go run` and pass a file as an argument. In the root project directory:

```bash
go run cmd/adventlang.go tests/__run_tests.adv
```

### An Example Program

```js
// An if statement
if (true) {}

// An assignment expression declaring and setting a variable
// to an Immediately Invoked Function Expression (IIFE)
let result = (func(x) { return x + 1 })(4);

// Implemention of a Set using a closure over a dictionary
// `let my_set = set();` or `let items = set([1, 2])`
let set = func(list) {
    let store = {};
    if (type(list) == "list") {
        for (let i = 0; i < 3; i = i + 1) {
            let key = list[i];
            store[key] = true;
        }
    }
    return {
        "add": func(x) { store[x] = true; },
        "has": func(x) { return store[x] == true }
    }
};

// An example of a computed key
let key = "a";
let f = {key: 2};

// A runtime assert call, used in test programs
assert(f.a, 2);
```

### Build

Build for common platforms:

```bash
./build.sh
```

### Run Tests

```bash
./run_tests.sh
```

### License

MIT.
