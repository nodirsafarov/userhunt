# Contributing to userhunt

Thanks for taking the time! Pull requests are welcome — especially new platforms, accuracy fixes, and detection improvements.

## Development setup

```bash
git clone https://github.com/nodirsafarov/userhunt
cd userhunt
make test     # run unit tests
make vet      # static analysis
make build    # produce ./bin/userhunt
```

## Adding a platform

1. Open `internal/platforms/platforms.json`.
2. Add a new entry:

   ```json
   {
     "name": "ExampleSite",
     "url": "https://example.com/u/{}",
     "category": "social",
     "check_type": "status"
   }
   ```

3. Pick `check_type`:
   - `status` — site returns `200` for existing users, `404` for missing. Use this when possible.
   - `content` — site always returns `200` (SPA, custom 404 page). Provide `not_exists_content` substrings that appear on missing-user pages, or `exists_content` substrings that appear only on real profiles.
4. Verify locally:

   ```bash
   make build
   ./bin/userhunt some-real-account-here --only-found
   ./bin/userhunt very-unlikely-name-zxq987 --only-found
   ```

   The first run must include your platform. The second must not.
5. Add a short note to the PR description explaining how you verified both directions.

## Coding standards

- Run `make vet` and `make test` before opening a PR.
- Keep the diff focused — one concern per PR.
- Public Go identifiers must have a doc comment starting with the identifier name (this is enforced by `revive`).
- Prefer the standard library; add new dependencies only when the value is clear.

## Reporting issues

Bug reports should include:

- The exact command that reproduces the problem.
- The username you searched (or a comparable one — please don't post sensitive targets).
- The platform that misbehaved, plus the URL and HTTP status code if you have them.

## Code of conduct

Be excellent to each other. Harassment of any kind is not tolerated in this repository.
