# API

## Jobs

POST /jobs/track/{id}
POST /jobs/album/{id}
POST /jobs/artist/{id}

GET /jobs
GET /jobs/{id}
DELETE /jobs/{id}

---

## Library

GET /artists
GET /albums/{id}
GET /tracks/{id}

---

## Behavior

All POST endpoints enqueue jobs.
They never block waiting for downloads.
