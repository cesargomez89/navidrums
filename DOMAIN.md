# Domain Model

## Track
Remote metadata entity describing a song.

Not guaranteed to exist locally.

---

## Download
A local file produced from a Track.

Has tags and filesystem path.

---

## Album
Collection of tracks grouped under a release.

---

## Artist
Collection of albums.

---

## Job
Represents a background task.

Types:
- track download
- album download
- artist download

State machine:
pending → processing → completed | failed | cancelled

---

## Provider
External music catalog source.

Providers do not persist state.

---

## Worker
Executes jobs asynchronously.

Workers never decide business rules.
They only execute service instructions.

