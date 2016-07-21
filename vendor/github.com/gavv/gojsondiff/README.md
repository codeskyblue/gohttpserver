# Go JSON Diff (and Patch)

## How to use

### Installation

```sh
go get github.com/yudai/gojsondiff
```

### Comparing two JSON strings

See `jd/main.go` for how to use this library.


## CLI tool

This repository contains a package that you can use as a CLI tool.

### Instllation

```sh
go get github.com/yudai/gojsondiff/jd
```

### Usage

#### Diff

Just give two json files to the `jd` command:

```sh
jd one.json another.json
```

Outputs would be something like:

```diff
 {
   "arr": [
     0: "arr0",
     1: 21,
     2: {
       "num": 1,
-      "str": "pek3f"
+      "str": "changed"
     },
     3: [
       0: 0,
-      1: "1"
+      1: "changed"
     ]
   ],
   "bool": true,
   "num_float": 39.39,
   "num_int": 13,
   "obj": {
     "arr": [
       0: 17,
       1: "str",
       2: {
-        "str": "eafeb"
+        "str": "changed"
       }
     ],
+    "new": "added",
-    "num": 19,
     "obj": {
-      "num": 14,
+      "num": 9999
-      "str": "efj3"
+      "str": "changed"
     },
     "str": "bcded"
   },
   "str": "abcde"
 }
```

When you prefer the delta foramt of [jsondiffpatch](https://github.com/benjamine/jsondiffpatch), add the `-f delta` option.

```sh
jd -f delta one.json another.json
```

This command shows:

```json
{
  "arr": {
    "2": {
      "str": [
        "pek3f",
        "changed"
      ]
    },
    "3": {
      "1": [
        "1",
        "changed"
      ],
      "_t": "a"
    },
    "_t": "a"
  },
  "obj": {
    "arr": {
      "2": {
        "str": [
          "eafeb",
          "changed"
        ]
      },
      "_t": "a"
    },
    "new": [
      "added"
    ],
    "num": [
      19,
      0,
      0
    ],
    "obj": {
      "num": [
        14,
        9999
      ],
      "str": [
        "efj3",
        "changed"
      ]
    }
  }
}
```

#### Patch

Give a diff file in the delta format and the JSON file to the `jp` command.

```sh
jp diff.delta one.json
```


## License

MIT License (see `LICENSE` for detail)
