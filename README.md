# goprotowrap

⚠️ **This is a maintenance fork of the [original repository] which is archived.**

[original repository]: https://github.com/square/goprotowrap

A package-at-a-time wrapper for protoc, for generating Go protobuf
code.

It is used in the [aperturerobotics/protobuf-project] template Makefile.

[aperturerobotics/protobuf-project]: https://github.com/aperturerobotics/protobuf-project/blob/995c779/Makefile#L76

`protowrap` is a small tool built by Square for compiling [Protobuf].

[Protobuf]: https://developers.google.com/protocol-buffers/

## Install

```shell
go get -u github.com/aperturerobotics/goprotowrap/cmd/protowrap
```

## Philosophy

`protowrap` is called instead of `protoc`, and ensures that `.proto` files are
processed one Go package at a time.

It parses out the flags it understands, passing the rest through to `protoc`
unchanged.

## Operation

- search for all `.proto` files under the same import paths as the
  `.proto` file arguments
- call `protoc` to generate FileDescriptorProtos
- inspect the FileDescriptorProtos to deduce package information
- group `.proto` files into packages
- call `protoc` once for each package

## License

Copyright 2016 Square, Inc.

Licensed under the Apache License, Version 2.0 (the "License"); you
may not use this file except in compliance with the License. You may
obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing
permissions and limitations under the License.
