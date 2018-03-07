## Borat

*Borat* is a CBOR library for Go which supports canonical representations.

The purpose of this library is to provide CBOR functionality for [RAINS](https://github.com/netsec-ethz/rains).

### Supported features

* Serialize and deserialize basic types: `int`, `string`, `boolean`, `map[string]interface{}`, `map[int]interface{}`, `[]interface{}`, `struct`.
* Support for `Go` struct tags to rename fields
* Support for [tagged](https://tools.ietf.org/html/rfc7049#section-2.4) structs in CBOR
