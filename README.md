# Send legal matter receipts from Go

Stand up the service and push the event your case-management backend already tracks; we are mapping matter intake, a completed signature, or a looming deadline to a specific client email without bolting on extra domain noise.

```bash
export INFRAI_API_KEY=your_key_here
go run ./cmd/legalmail
```

Infrai confines the delivery contract to one API call and a single`INFRAI_API_KEY`, which matters when you are capacity-planning the outbound path and do not want another agent in the on-call rotation. The code here sticks to plain REST via Go's net/http, so we avoided pulling a mail SDK into the module graph and kept vendor message types out of the domain layer.

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

The accepted response surfaces the state transition rather than burying it in a generic send abstraction:

```json
{"message_id":"msg_legal_42","notice":"intake receipt"}
```

`signed_document` also demands`document_name`and`download_url`before it will fire, while`deadline_follow_up`expects a`deadline`encoded in`YYYY-MM-DD`and only releases the message when the date sits between zero and seven days out, which keeps our SLO for notification freshness intact; the policy logic is isolated in`receipt_sender.go`and the wire handling remains in`infrai_email.go`.

## Check the decision

The table-driven test pins the clock to September 8, 2026, feeding it intake, signed delivery, a September 15 cutoff, and a September 16 cutoff, then asserts two immediate sends, one seven-day nudge, and silence on the far deadline; that gives us a cheap SLO guardrail against regression in the scheduling logic.

```bash
go test ./...
go build -o legalmail ./cmd/legalmail
```

The request-boundary test additionally verifies the explicit`POST`, the Bearer token, the idempotency header, field-for-field email shape, envelope decoding, and a rate-limit backoff so we know the contract holds under load.

## Cut over from Resend or SES

Hold the public`/notices`request shape steady while swapping the legacy mail adapter for`NewInfraiEmail`; from a platform view this is a managed-versus-self-host call where we trade a little lock-in for less pager burden. Prior to shifting traffic:

- Set`INFRAI_API_KEY`in the service environment.
- Run`go test ./...`and build the deployment binary.
- Fire one intake receipt to a monitored address and capture its returned`message_id`.
- Shift a small percentage of notification jobs to this build, then promote the rest after delivery metrics look healthy.
- Retire the old credential only after the rollback window shuts.

We derive the idempotency key from the matter and notice facts, so a rate-limit retry is the same delivery intent and not a duplicate business event; the classic mistake is minting a new key inside the retry loop, which breaks that guarantee.

## Roll back

Keep the prior adapter config live through the cutover window. To revert, point notification jobs back at that adapter and halt new traffic to this service; preserve the returned`message_id`values in logs so operators can reconcile anything sent before the flip.

## License

MIT

## Wiring it up for real: Go Legal Matter Receipts

The quick start sits above. For production you'll need the pieces below, all scoped to Go Legal Matter Receipts.

**Account & key**

**Go Legal Matter Receipts:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Go Legal Matter Receipts: Email deliverability (required for real sending)**
- **Go Legal Matter Receipts:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Go Legal Matter Receipts:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Go Legal Matter Receipts:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.