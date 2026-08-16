// A minimal custom CodeMirror language for raw RFC 5322 (.eml) documents
// (SPEC.md §7.7.4.1: "no off-the-shelf .eml grammar exists"). Highlights
// header lines, the blank-line header/body boundary, and MIME part
// boundaries — not a full RFC 5322 grammar, just enough for the Composer's
// editor to visually distinguish the parts a template author cares about.

import { StreamLanguage } from '@codemirror/language'
import type { StreamParser } from '@codemirror/language'

interface EmlState {
  inBody: boolean
}

const headerLineRe = /^[\w-]+:/
const boundaryRe = /boundary=/i

const emlStreamParser: StreamParser<EmlState> = {
  startState(): EmlState {
    return { inBody: false }
  },

  token(stream, state) {
    if (stream.sol() && stream.eol()) {
      // A blank line marks the header/body boundary the first time it's seen.
      state.inBody = true
      return null
    }

    if (!state.inBody) {
      if (stream.sol()) {
        if (stream.match(headerLineRe)) {
          return 'keyword'
        }
        // A continuation line (folded header) — still header territory.
        if (stream.match(/^\s+/)) {
          return null
        }
      }
      if (boundaryRe.test(stream.string)) {
        stream.skipToEnd()
        return 'string'
      }
      stream.next()
      return null
    }

    // Body: highlight template placeholders and MIME boundary markers so
    // they stand out from plain prose.
    if (stream.match(/\{\{.*?\}\}/)) {
      return 'variableName'
    }
    if (stream.sol() && stream.match(/^--\S+/)) {
      return 'string'
    }
    stream.next()
    return null
  },
}

export const emlLanguage = StreamLanguage.define(emlStreamParser)
