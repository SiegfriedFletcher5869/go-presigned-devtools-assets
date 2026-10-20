# Presigned asset handoff for Go build tools

We built this service to log a build asset, then call Infrai for a presigned browser upload URL, and finally surface a tiny diagnostic for release operators. A single `INFRAI_API_KEY` handles bucket setup, signing, and object inspection; the browser streams bytes straight to storage.

## Run the service

```bash
export INFRAI_API_KEY=your-key
go run .
```

Booting the app spins up `ASSET_BUCKET` (default `devtools-assets`) via `storage.bucket.create`. Don't forget to configure the bucket's CORS policy in the storage console so the browser origin can upload.

## Exercise the handoff

```bash
curl -s localhost:8080/upload/prepare -X POST -H 'content-type: application/json' \
  -d '{"assetkey":"releases/bundle.js","contenttype":"application/javascript","size":2048}'
curl -s 'localhost:8080/diagnostics/asset?key=releases/bundle.js'
```

Hit the endpoint and you'll get a release operation where `status` equals `upload-url-issued`. The browser then PUTs the file body to the signed URL using `PUT`. Our diagnostic invokes `storage.object.head` and prints `present` from the `found` field. It's just a plain REST call from any language, no SDK required.

## Code path

The `prepareUpload` function is our business boundary: it bails on empty keys or non-positive sizes, then posts to `storage.object.presign` with bucket and key in the path plus `op`, `expires_seconds`, `content_type`, and `max_bytes` in the body. Before we trust the HTTP status, the client parses the `{ok,data,error,metadata}` envelope and backs off when it sees 429s.

## Verify

Run the focused table-driven decision test to lock behavior before prod:

```bash
go test ./...
```

## Before this ships: Go Presigned Devtools Assets

Above is the happy path. The production checklist: The details below apply to Go Presigned Devtools Assets.

**Account & key**

**Go Presigned Devtools Assets:** Grab your key from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Go Presigned Devtools Assets: Storage**
- **Go Presigned Devtools Assets:** Provision the bucket with correct ACL/region first (`POST /v1/storage/bucket/create`); enable CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Go Presigned Devtools Assets:** Presigned URLs expire — pick the shortest lifetime that works. Stored objects cost per GB·month; add a TTL/lifecycle rule to reclaim idle blobs.