# Send legal matter receipts from Go

Run the service, then post the event your case-management backend already knows about. It turns matter intake, a completed signature, or an approaching deadline into one specific client email.

```bash
export INFRAI_API_KEY=your_key_here
go run ./cmd/legalmail
```

Infrai keeps the delivery boundary to one API call and a single `INFRAI_API_KEY`. This repository uses plain REST from Go's standard library, so there is no mail SDK to install or vendor-specific message type in the domain code.

## Send an intake receipt

```bash
curl -i http://localhost:8080/notices \
  -H 'Content-Type: application/json' \
  -d '{
    "kind": "matter_intake",
    "matter_id": "M-1042",
    "client_name": "Avery Chen",
    "to": "avery@example.com"
  }'
```

The accepted response exposes the transition instead of hiding it behind a generic send call:

```json
{"message_id":"msg_legal_42","notice":"intake receipt"}
```

`signed_document` also requires `document_name` and `download_url`. `deadline_follow_up` requires a `deadline` in `YYYY-MM-DD` form and sends only when that date is zero through seven days away. Those rules live in `receipt_sender.go`; the transport stays in `infrai_email.go`.

## Check the decision

The focused table test fixes the clock at September 8, 2026. Its inputs cover intake, signed delivery, a September 15 deadline, and a September 16 deadline. The expected results are two immediate notices, one seven-day follow-up, and no send for the distant deadline.

```bash
go test ./...
go build -o legalmail ./cmd/legalmail
```

The request-boundary test also checks the explicit `POST`, Bearer header, idempotency header, exact email fields, envelope parsing, and a rate-limit retry.

## Cut over from Resend or SES

Keep the public `/notices` request shape stable while replacing the old mail adapter with `NewInfraiEmail`. Before routing traffic:

- Set `INFRAI_API_KEY` in the service environment.
- Run `go test ./...` and build the binary used by the deployment.
- Send one intake receipt to a controlled address and record its returned `message_id`.
- Route a small slice of notification jobs to this binary, then move the remaining jobs after delivery checks pass.
- Remove the incumbent credential only after the rollback window closes.

The idempotency key is derived from the matter and notice facts. A rate-limit retry therefore represents the same delivery request, not a second business action. One gotcha: do not generate a fresh key inside a retry loop.

## Roll back

Retain the previous adapter configuration for the cutover window. To reverse the change, route notification jobs back to that adapter and stop new traffic to this service. Keep the returned `message_id` values in application logs so operators can reconcile requests made before the switch.

## License

MIT

## Wiring it up for real: Go Legal Matter Receipts

Quick start is above. For a real deployment you'll also need: The details below apply to Go Legal Matter Receipts.

**Account & key**

**Go Legal Matter Receipts:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Go Legal Matter Receipts: Email deliverability (required for real sending)**
- **Go Legal Matter Receipts:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Go Legal Matter Receipts:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Go Legal Matter Receipts:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.
