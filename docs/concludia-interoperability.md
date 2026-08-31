# Concludia Interoperability

## Goal

A workspace is a permissive environment for inquiry. Concludia is the target
modality for a rooted argument suitable for publication and contestation.

The boundary is explicit export:

```text
workspace + selected root -> complete rooted structure -> Concludia `.arg`
```

The whole workspace is not implicitly uploaded or imported.

## Reading Concludia files

The tool must open an ordinary Concludia `.arg` as a workspace without semantic
loss. This includes:

- connected graphs with one or more conclusions;
- `AND` and `OR` junctors;
- legacy direct supports;
- undermines and undercuts;
- counterpoints targeting counterpoints;
- labels, slugs, truth tokens, kinds, and metadata.

Statement declaration order is preserved as the file's general presentation
order. Cludia may change it only through an explicit reorder operation; the
operation does not change logical relations.

Focused editing may be narrower than the representable format. Unsupported
focused creation does not authorize dropping or rewriting existing constructs.

The stable permissive marker is `profile="cludia"`. Pre-1.0
`profile="workspace"` is accepted as an input alias and migrates only on a
successful Cludia save. Rooted export removes the permissive profile marker.
Optional `meta version` remains graph/artifact metadata: Cludia preserves an
authored value but does not invent one for new workspaces or exports.

Concludia's JSON model distinguishes opaque server step identities from
display labels. Its human-oriented `.arg` DSL substitutes labels into the
declaration and relation positions, so those labels act as portable file-local
IDs. Cludia continues to model this portable layer rather than adding an opaque
identity field that Concludia's DSL does not carry.

Ordinary Cludia allocation keeps these local IDs monotonic. An explicit
`renumber` is a reviewed file-level identity migration and produces a complete
mapping; it is not equivalent to Concludia changing labels over unchanged
server identities. Stable cross-system provenance remains future integration
work if incremental synchronization is later required.

## Rooted export algorithm

Given selected root statement `R`:

1. Verify that `R` exists and is not a counterpoint.
2. Add `R` to the result.
3. For every support or junctor targeting an included non-counterpoint
   statement, include the complete relation and all of its sources.
4. Repeat step 3 recursively for newly included sources. Include every incoming
   justification; do not choose a preferred proof path.
5. For every included premise or other valid statement target, include all
   defeats attached to that statement.
6. For every included junctor, include all inference-scope defeats attached to
   that junctor at its included target.
7. For each included counterpoint, include:
   - its complete upstream justifications, if any;
   - every counterpoint targeting it;
   - the same information recursively for those counterpoints.
8. Repeat steps 3 through 7 to a fixed point whenever any statement, junctor,
   support, or defeat is newly included. In particular, sources introduced by
   a counterpoint's justification receive their own complete support closure
   and attached defeat chains.
9. Exclude workspace statements and components that are not reached by these
   typed traversal rules. Do not follow ordinary downstream support from the
   selected root into unrelated conclusions.
10. Emit included statements in their general workspace order.
11. Reconcile statement roles.
12. Set the selected root in export metadata.
13. Remove Cludia-owned allocator metadata such as `cludia-next-ids`.
14. Remove Cludia-profile metadata.
15. Validate under the Concludia profile.
16. Write the output atomically only if validation succeeds.

This is the **entire rooted structure**: all available upstream justifications
and their attached contestation, rather than one selected path.

Reachable undercut junctors remain in the export until the author explicitly
removes them. A repair that leaves the original junctor in the workspace does
not silently exclude it. Concludia-profile validation checks their structure,
not whether their sufficiency claims have survived semantic challenge.

## Role reconciliation

For the exported structure:

- selected root -> `conclusion`;
- non-counterpoint statement targeted by support -> `lemma`;
- unsupported non-counterpoint leaf -> `premise`;
- every counterpoint -> `counterpoint`.

Reconciliation must preserve local IDs where the Concludia profile permits.
Any label normalization must be deterministic and reported.

Truth handling follows the shared `.arg` rules. Export must not invent a
confidence value or silently assert unknown ground statements as true.

Propagated effective truth is not serialized into rooted exports. Export keeps
authored leaf tokens and support/defeat structure; Cludia and Concludia can
recalculate the grounded overlay. Legacy sourced truth remains preserved unless
the user explicitly applies Cludia's normalization repair before export.

## Defeats and semantic validity

Defeats are part of the exported argument, not reasons to omit the challenged
structure. A conclusion may therefore export as structurally valid while
remaining contested.

Structural export success means:

- the file parses;
- references and roles are valid;
- the rooted graph satisfies Concludia's topology;
- defeat targets are well-formed.

It does not mean every premise is true or every junctor semantically entails
its target. Those remain subject to Concludia's semantic validation process.
Calculated effective truth describes the structural three-valued consequence of
the current authored leaves and grounded defeat overlay; it is not a semantic
proof of the natural-language claims.

For human presentation, Cludia follows Concludia's role-aware notation:
effective `T/U/F` on lemmas and conclusions appears as `⊢/◇/⊬`. This is a
presentation of the same calculated overlay, not a file-format conversion.
`⊬ P` means the current argument does not prove `P`, not that it proves a
separate negation of `P`.

## Diagnostics

Failed export should identify actionable issues such as:

- missing source or target;
- cycle;
- invalid defeat scope or target;
- premise that requires promotion;
- isolated element unexpectedly pulled into the closure;
- unsupported syntax or lossy serialization risk;
- Concludia-profile connectivity or root failure.

Diagnostics must use stable codes in JSON. Failure must not leave a partial
output file.

## Compatibility testing

The new repository should maintain fixtures representing:

- a disconnected workspace;
- the Concludia export of a selected root;
- an ordinary Concludia graph containing `OR`;
- a graph containing legacy direct support;
- premise, inference, and recursive counterpoint defeats;
- a contested junctor coexisting with a repaired replacement;
- multiple justifications into one target;
- multiple conclusions sharing premises;
- invalid cycles and dangling references.

If Concludia and the new tool evolve separately, a small shared conformance
corpus is preferable to copying implementation assumptions between Scala and
Go.

### Current supported-counterpoint validation edge

Concludia's current validator permits a counterpoint to have upstream support,
but a counterpoint targeted by another counterpoint must be a support leaf.
Consequently, a supported counterpoint that is itself counterpointed is valid
workspace structure but currently fails the strict Concludia profile. Cludia
must preserve the structure and report
`concludia_defeat_target_not_leaf`; it must not drop either the support or the
recursive counterpoint. Rooted export diagnoses and refuses that combination
until the Concludia interoperability edge is resolved.

## Future integration

Possible later integrations include:

- importing a rooted closure directly through Concludia MCP;
- opening a Concludia graph as a new local inquiry workspace;
- promoting a connected component without writing an intermediate file;
- preserving stable cross-system IDs and provenance.

These must remain explicit operations. A local exploratory edit must never
silently mutate a published Concludia graph.
