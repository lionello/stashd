#!/usr/bin/env rdmd
// Get rdmd from https://dlang.org/download.html (or brew, nix, apt, yum)
// Copyright Lionello Lunesu. Placed in the public Domain.
// https://gist.github.com/lionello/84cad70f835131198fee4ab7e7592fce

import std.stdio : writeln;

int main(string[] args) {
  import std.process : pipeProcess, wait, Redirect, environment, pipeShell;
  import std.stdio : stdout;

  if (args.length < 2) {
    return usage(args[0]);
  }

  // Check if PAGER is set
  const pager = environment.get("PAGER");
  if (pager != "") {
    writeln("Using pager: ", pager);
    // pipe our own output through the pager
    auto pipes = pipeShell(pager, Redirect.all);
  }

  // FIXME: support proper command line arguments
  const filenames = args[1..$];
  auto extraArgs = detectTerminal() ? ["--color=always"] : [];
  foreach (arg; filenames) {
    if (arg == "--") break;
    if (arg == "-h" || arg == "--help") return usage(args[0]);
    if (arg.length >= 2 && arg[0] == '-') extraArgs ~= arg;
  }

  auto pipes = pipeProcess(["git", "stash", "list", "-p"] ~ extraArgs, Redirect.all);

  enum STASH = "stash@{";
  enum DIFF = "diff --git a/";

  string lastStash = null;
  bool dump = false;
  foreach (line; pipes.stdout.byLine) {
    if (line.length > 10) {
      // Find the start of the filenames by skipping ANSI color and the diff prefix
      const start = line[0] == '\x1b' ? 4 : 0;
      const end = start + DIFF.length;
      if (line.length > end && line[start..end] == DIFF) {
        // Encountered a new diff; stop dumping and check filenames
        dump = false;
        if (anyMatch(filenames, line[end..$])) {
          // Found a match; print stash header if not yet printed
          if (lastStash) {
            writeln(lastStash);
            writeln();
            lastStash = null;
          }
          dump = true;
        }
      } else if (line[0..7] == STASH) {
        // Encountered a new stash; stop dumping but save header in case a file matches
        dump = false;
        lastStash = line.idup;
      }
    }
    if (dump) {
      writeln(line);
    }
  }
  // stdout.flush();
  // stdout.close();
  return wait(pipes.pid);
}

@nogc @safe pure nothrow
private bool anyMatch(in char[][] names, in char[] line) {
  import std.string : indexOf;
  foreach (name; names) {
    if (line.indexOf(name) >= 0) {
      return true;
    }
  }
  return false;
}

private int usage(string arg0) {
  import std.path : baseName;
  writeln("Usage: ", arg0.baseName, " [<diff options>] [--] filename...");
  return 129;
}

// Inspired by DMD's src/dmd/console.d
private bool detectTerminal() {
  import std.process : environment;
  import core.sys.posix.unistd : isatty, STDOUT_FILENO;
  const term = environment.get("TERM");
  return isatty(STDOUT_FILENO) && term && term != "dumb";
}
