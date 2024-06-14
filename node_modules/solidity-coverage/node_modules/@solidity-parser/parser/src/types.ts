export interface Node {
  type: string
}

export interface TokenizeOptions {
  range?: boolean
  loc?: boolean
}

export interface ParseOptions extends TokenizeOptions {
  comments?: boolean
  tokens?: boolean
  tolerant?: boolean
}

export interface Token {
  type: string
  value: string | undefined
  range?: [number, number]
  loc?: {
    start: {
      line: number
      column: number
    }
    end: {
      line: number
      column: number
    }
  }
}
