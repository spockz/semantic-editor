<!-- WHY: Keep less frequent, multi-step Java migration recipes available on demand without loading them during routine Java operation routing. -->
# Advanced Java Refactoring Recipes

Read this reference for less frequent interface/contract, signature, parameter, declaration-movement, callable-inlining, Make Static, and class-shape transformations. The [Java language guide](../java.md) contains routine operation routing and backend boundaries.

## Change signatures and introduce parameters

Use `refactor_change_signature` with one signature-qualified Java method `symbol` and the complete ordered `parameters` mapping. Preserve each existing parameter with its `source_index`; added parameters require a closed `argument` expression so JDT LS can update every resolved call site. The returned workspace edit is validated and applied across affected Java files, then each file is formatted and checked for diagnostics. Keep the signature change and caller updates in one operation.

Use `refactor_introduce_parameter` when one exact expression in a Java method should become a new parameter. Supply the complete ordered `parameters` mapping, preserve existing entries by `source_index`, add the new parameter with its call-site `argument`, and identify the expression with `discriminator` plus `construct_path` when needed. JDT LS updates the method signature and resolved callers; semedit then replaces the selected expression in the method body and verifies every changed file.

## Extract interfaces and implement contracts

Use `refactor_implement_contract` with the concrete class `file` and `symbol`, interface name in `contract`, and its source in optional `contract_file`. The operation adds the interface when absent, then uses JDT LS `java/listOverridableMethods` and `java/addOverridableMethods` to generate missing method stubs. Java currently writes to the selected class file, so omit `destination_file` or set it equal to `file`. Replace generated placeholders with `semantic_replace_body` before treating them as implemented behavior.

Use `refactor_extract_interface` with the concrete type in `in_type`, a new interface `symbol`, and exact direct method selectors in `members`. Java extracts public instance methods only, appends a package-private interface to the same source file, adds it to the class, then formats and verifies through JDT LS. Omit `destination_file` or set it equal to `file`; overload selectors must identify one method signature.

## Introduce a parameter object

Use `refactor_introduce_parameter_object` when a Java method's full parameter list should become one cohesive value object. Supply a `parameter` with the new argument name and type, map every original argument once through `fields` in `source_index` order, and select an existing same-file `destination_file`. The operation generates a package-private immutable class with a constructor and accessors, asks JDT LS to replace the method parameters and update references, then rewrites each constructor call with its original arguments in order and replaces method-body parameter uses with accessors. It refuses public or protected methods, cross-package callers, annotated or otherwise unsupported parameter syntax, argument comments, field type changes, non-invocation references, incomplete call-site edits, and stale source revisions. JDT LS formats and checks every changed file before publication.

## Move a declaration

Use `refactor_move_declaration` to move a Java static method, static field, or static nested type to an existing destination type. Supply the source `file` and `symbol`, destination type's `destination_file`, and its qualified name as `target_symbol`; omit Go placement options. JDT LS performs the move, updates references and imports across affected files, and semedit verifies the resulting workspace edit. Instance methods and non-static nested types refuse.

The same operation can move a public top-level type to an existing destination file in another package. Set `file` to the current type file, `symbol` to the type name, and `destination_file` to an existing target type in the destination package; `target_symbol` selects that target. JDT LS may return a resource rename for the source `.java` file, which semedit validates together with the workspace edits. The move must remain inside the trusted workspace, the new file path must be available in the destination package, and buffered batches refuse resource renames.

## Inline a callable

Use `refactor_inline_callable` to inline one Java invocation. Select the callee with `file` and `symbol`; use `call_site` to select the caller file, its `in_function`, and the exact call expression as `discriminator`. The expression must resolve uniquely in the method, and JDT LS `refactor.inline` edits only that selected caller file.

## Make a method static

Use `refactor_make_static` with a trusted Java workspace, a selected source `file`, and an exact signature-qualified method `symbol` from `semantic_lookup` or `semantic_inspect_symbol`. The method's simple name must identify exactly one method in the selected file; overloads and same-name methods in another class in that file are refused without edits. Semedit accepts only one enabled `Make Static` action. The native workspace edit may add an explicit receiver parameter and update call sites in other files. Semedit validates and publishes all affected files, then resolves the post-edit method identity before returning its revision-bound receipt continuation.

## Change class shape

### Modernize a class to a record

When migrating an immutable data carrier class to a Java 16+ `record`, inspect the class with `semantic_outline` and `semantic_inspect_symbol`. Confirm that fields are `private final`, state is immutable, the class has no external superclass, and serialization/reflection conventions permit records. For example:

```java
public final class Point {
    private final int x;
    private final int y;
    public Point(int x, int y) { this.x = x; this.y = y; }
    public int getX() { return x; }
    public int getY() { return y; }
}

public record Point(int x, int y) {}
```

Remove redundant canonical constructors, standard getters, `equals`, `hashCode`, and `toString`; preserve validation in a compact constructor. Update JavaBean getter call sites to record component accessors. Use `refactor_rename` where its symbol contract fits. Run `semantic_verify` with formatting and import organization, then `semantic_maven_compile` with workspace trust.

### Expand a record to a class

When a record needs mutable caches, lazy initialization, or inheritance, inspect its components and invariants with `semantic_lookup` and `semantic_inspect_symbol`. Replace it with a class containing explicit fields, constructor, component accessors, and behaviorally equivalent `equals`, `hashCode`, and `toString`. Preserve existing component accessor methods such as `point.x()`. Format with `semantic_verify` and compile via `semantic_maven_compile`.

### Encapsulate a field

For an exposed field, inspect accesses and annotations with `semantic_lookup` and `semantic_inspect_symbol`. Add explicit accessors, change visibility to `private`, then rewrite each field read or assignment as the matching method invocation using a language-aware edit; `refactor_rename` renames symbols but cannot turn field access into getter/setter calls. Java has no dedicated field-encapsulation operation. Format with `semantic_verify` and run `semantic_maven_test`.
