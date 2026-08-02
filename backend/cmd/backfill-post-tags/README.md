# Historical post tag backfill

The command reads all non-deleted posts, preserves existing tags, adds only
controlled AI topic tags, and always derives the subject-combination tag from
`track` and `electives`.

It is a dry-run unless `-execute` is supplied. Write the audit report outside
the repository and back up the database before applying it:

```sh
go run ./cmd/backfill-post-tags -report /tmp/post-tag-backfill.json
go run ./cmd/backfill-post-tags -execute -report /tmp/post-tag-backfill-applied.json
```

`-limit` can be used for a small pilot. AI failures are recorded per post and
do not prevent the deterministic combination tag from being applied. Re-running
the command is idempotent for tags.
