#### Description

The `computer` command group is computer-use for AI agents: seeing the screen and acting on it. The generic
commands (`aux4 computer <command>`) come from `aux4/computer` and dispatch to the provider for the current
operating system; this package adds the Linux provider under `aux4 computer linux`.

#### Usage

```bash
aux4 computer linux <command> [options]
```

#### Example

```bash
aux4 computer linux info
```
