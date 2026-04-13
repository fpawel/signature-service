# Signature Service

A small HTTP service for creating signature devices and using them to sign transaction data.

Each device has:

- a unique ID
- a signing algorithm (`ECC` or `RSA`)
- an optional label
- an internal `signature_counter`
- the `last_signature` value used to chain signatures

For every signing request, the service signs the following value:

```text
<signature_counter>_<data_to_be_signed>_<last_signature_base64_encoded>
```

For the very first signature of a device, `last_signature` is the base64-encoded device ID.

## Current scope

This implementation currently provides:

- `POST /devices` to create a signature device
- `GET /devices` to list signature devices
- `GET /devices/{deviceId}` to retrieve a single signature device
- `POST /devices/{deviceId}/signatures` to create a signature
- `GET /devices/{deviceId}/signatures` to list signatures for a device
- `GET /devices/{deviceId}/signatures/{signatureId}` to retrieve a single signature

Storage is in memory only, so all data is lost when the process stops.

---

## Prerequisites

- Go 1.20+
- `make`
- `curl`

Optional for regeneration of generated Swagger code:

- `swagger` CLI

---

## How to run the server

### Option 1: using Make

```bash
make run
```

This starts the server on:

```text
http://localhost:8080
```

### Option 2: using Go directly

```bash
go run ./cmd/server/... 8080
```

The server expects the port as a positional argument.

---

## API documentation

Swagger UI is available at:

```text
http://localhost:8080/docs
```

---

## Quick manual QA with curl

Open a second terminal and use the commands below while the server is running.

### Test variables

```bash
BASE_URL=http://localhost:8080
ECC_DEVICE_ID=11111111-1111-4111-8111-111111111111
RSA_DEVICE_ID=22222222-2222-4222-8222-222222222222
UNKNOWN_DEVICE_ID=33333333-3333-4333-8333-333333333333
SIGNATURE_ID=<paste-a-signature-id-from-the-list-response>
```

---

## 1. Create an ECC signature device

```bash
curl -i \
  -X POST "$BASE_URL/devices" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "'"$ECC_DEVICE_ID"'",
    "algorithm": "ECC",
    "label": "Main ECC device"
  }'
```

Expected result:

- HTTP status: `201 Created`
- response contains:
  - `id`
  - `public_key_pem`

---

## 2. Create an RSA signature device

```bash
curl -i \
  -X POST "$BASE_URL/devices" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "'"$RSA_DEVICE_ID"'",
    "algorithm": "RSA",
    "label": "Main RSA device"
  }'
```

Expected result:

- HTTP status: `201 Created`

---

## 3. Try to create the same device again

```bash
curl -i \
  -X POST "$BASE_URL/devices" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "'"$ECC_DEVICE_ID"'",
    "algorithm": "ECC",
    "label": "Duplicate device"
  }'
```

Expected result:

- HTTP status: `409 Conflict`

---

## 4. Create the first signature for the ECC device

```bash
curl -i \
  -X POST "$BASE_URL/devices/$ECC_DEVICE_ID/signatures" \
  -H "Content-Type: application/json" \
  -d '{
    "data": "payment-001"
  }'
```

Expected result:

- HTTP status: `201 Created`
- response contains:
  - `signature`
  - `signed_data`

For the first signature, `signed_data` should start with:

```text
0_payment-001_
```

For the test device ID above, the base64-encoded device ID is:

```text
MTExMTExMTEtMTExMS00MTExLTgxMTEtMTExMTExMTExMTEx
```

So the full expected `signed_data` value is:

```text
0_payment-001_MTExMTExMTEtMTExMS00MTExLTgxMTEtMTExMTExMTExMTEx
```

---

## 5. Create the second signature for the same device

```bash
curl -i \
  -X POST "$BASE_URL/devices/$ECC_DEVICE_ID/signatures" \
  -H "Content-Type: application/json" \
  -d '{
    "data": "payment-002"
  }'
```

Expected result:

- HTTP status: `201 Created`
- `signed_data` should start with:

```text
1_payment-002_
```

The last part of `signed_data` should now be the previous signature value, base64-encoded by the service response.

---

## 6. Try to sign with an unknown device

```bash
curl -i \
  -X POST "$BASE_URL/devices/$UNKNOWN_DEVICE_ID/signatures" \
  -H "Content-Type: application/json" \
  -d '{
    "data": "payment-404"
  }'
```

Expected result:

- HTTP status: `404 Not Found`

---

## 7. Try to create a device with an invalid algorithm

```bash
curl -i \
  -X POST "$BASE_URL/devices" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "44444444-4444-4444-8444-444444444444",
    "algorithm": "DSA",
    "label": "Invalid algorithm"
  }'
```

Expected result:

- HTTP status: `400 Bad Request`

---

## 8. Try to sign with an empty payload value

```bash
curl -i \
  -X POST "$BASE_URL/devices/$ECC_DEVICE_ID/signatures" \
  -H "Content-Type: application/json" \
  -d '{
    "data": ""
  }'
```

Expected result:

- HTTP status: `400 Bad Request`

---

## 9. Try to use an invalid device ID format in the path

```bash
curl -i \
  -X POST "$BASE_URL/devices/not-a-uuid/signatures" \
  -H "Content-Type: application/json" \
  -d '{
    "data": "payment-invalid-id"
  }'
```

Expected result:

- HTTP status: `400 Bad Request`

---

## 10. Helpful one-line smoke test sequence

```bash
BASE_URL=http://localhost:8080
DEVICE_ID=55555555-5555-4555-8555-555555555555

curl -sS -X POST "$BASE_URL/devices" \
  -H "Content-Type: application/json" \
  -d '{"id":"'"$DEVICE_ID"'","algorithm":"ECC","label":"Smoke test device"}'

echo

curl -sS -X POST "$BASE_URL/devices/$DEVICE_ID/signatures" \
  -H "Content-Type: application/json" \
  -d '{"data":"smoke-test-1"}'

echo

curl -sS -X POST "$BASE_URL/devices/$DEVICE_ID/signatures" \
  -H "Content-Type: application/json" \
  -d '{"data":"smoke-test-2"}'

echo
```

---
## 11. List all devices

```bash
curl -i   -X GET "$BASE_URL/devices"
```

Expected result:

- HTTP status: `200 OK`
- response contains a list of previously created devices

---

## 12. Get a single device

```bash
curl -i   -X GET "$BASE_URL/devices/$ECC_DEVICE_ID"
```

Expected result:

- HTTP status: `200 OK`
- response contains the selected device
- response should include at least:
  - `id`
  - `algorithm`
  - `label`
  - `signature_counter`
  - `last_signature`

---

## 13. List signatures for a device

Run this after creating at least one signature for the device.

```bash
curl -i   -X GET "$BASE_URL/devices/$ECC_DEVICE_ID/signatures"
```

Expected result:

- HTTP status: `200 OK`
- response contains signatures created with this device
- signatures should be returned in creation order

If the response contains signature IDs, copy one of them into `SIGNATURE_ID`.

Example:

```bash
SIGNATURE_ID=<paste-a-signature-id-from-the-response>
```

---

## 14. Get a single signature

```bash
curl -i   -X GET "$BASE_URL/devices/$ECC_DEVICE_ID/signatures/$SIGNATURE_ID"
```

Expected result:

- HTTP status: `200 OK`
- response contains one signature record
- response should include at least:
  - `id`
  - `device_id`
  - `counter`
  - `signature`
  - `signed_data`

---

## 15. Try to get a device that does not exist

```bash
curl -i   -X GET "$BASE_URL/devices/$UNKNOWN_DEVICE_ID"
```

Expected result:

- HTTP status: `404 Not Found`

---

## 16. Try to list signatures for an unknown device

```bash
curl -i   -X GET "$BASE_URL/devices/$UNKNOWN_DEVICE_ID/signatures"
```

Expected result:

- HTTP status: `404 Not Found`

---

## 17. Try to get a signature with an invalid signature ID format

```bash
curl -i   -X GET "$BASE_URL/devices/$ECC_DEVICE_ID/signatures/not-a-uuid"
```

Expected result:

- HTTP status: `400 Bad Request`

---


## Notes

- The service uses in-memory storage only.
- Restarting the process resets all devices and counters.
- The server returns JSON responses.
- The Swagger-generated documentation is useful for quick manual inspection at `/docs`.

---

## Development helpers

Regenerate Swagger server code:

```bash
make gen
```

Run the server:

```bash
make run
```
