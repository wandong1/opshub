/**
 * Variable Extraction Utility
 * Extracts {{variable}} patterns from text and filters out preset variables
 */

// System preset variables to exclude from extraction
const PRESET_VARIABLES = new Set([
  // Time-related
  'timestamp',
  'timestamp_ms',
  'current_time',
  'current_date',
  'current_datetime',
  // Random-related
  'random_number',
  'random_string',
  'random_uuid',
  // Inspection-related
  'exec_node_ip',
  'instance'
])

// Dynamic preset variable patterns (regex)
const DYNAMIC_PRESET_PATTERNS = [
  /^.+_instance$/ // Matches {label}_instance pattern
]

/**
 * Check if a variable name is a preset variable
 */
function isPresetVariable(varName: string): boolean {
  // Check static presets
  if (PRESET_VARIABLES.has(varName)) {
    return true
  }

  // Check dynamic patterns
  return DYNAMIC_PRESET_PATTERNS.some(pattern => pattern.test(varName))
}

/**
 * Extract variables from a single text string
 */
function extractFromText(text: string): Set<string> {
  const variables = new Set<string>()

  if (!text) {
    return variables
  }

  // Match {{variable}} pattern
  const regex = /\{\{([^}]+)\}\}/g
  let match

  while ((match = regex.exec(text)) !== null) {
    const varName = match[1].trim()

    // Skip empty or preset variables
    if (varName && !isPresetVariable(varName)) {
      variables.add(varName)
    }
  }

  return variables
}

/**
 * Extract variables from probe configuration
 */
export function extractFromProbeConfig(config: any): string[] {
  const variables = new Set<string>()

  if (!config) {
    return []
  }

  // Extract from basic fields
  const basicFields = [
    'target',      // 目标地址（所有类型）
    'port',        // 端口（TCP/UDP）
    'url',         // URL（HTTP/HTTPS/WebSocket）
    'body',        // 请求体（HTTP）
    'proxyUrl',    // 代理地址
    'wsMessage'    // WebSocket 消息
  ]

  basicFields.forEach(field => {
    if (config[field]) {
      extractFromText(config[field]).forEach(v => variables.add(v))
    }
  })

  // Extract from headers (JSON object or string)
  if (config.headers) {
    if (typeof config.headers === 'string') {
      try {
        const headersObj = JSON.parse(config.headers)
        Object.values(headersObj).forEach(value => {
          if (typeof value === 'string') {
            extractFromText(value).forEach(v => variables.add(v))
          }
        })
      } catch {
        // If not valid JSON, treat as plain text
        extractFromText(config.headers).forEach(v => variables.add(v))
      }
    } else if (typeof config.headers === 'object') {
      Object.values(config.headers).forEach(value => {
        if (typeof value === 'string') {
          extractFromText(value).forEach(v => variables.add(v))
        }
      })
    }
  }

  // Extract from params (JSON object or string)
  if (config.params) {
    if (typeof config.params === 'string') {
      try {
        const paramsObj = JSON.parse(config.params)
        Object.values(paramsObj).forEach(value => {
          if (typeof value === 'string') {
            extractFromText(value).forEach(v => variables.add(v))
          }
        })
      } catch {
        extractFromText(config.params).forEach(v => variables.add(v))
      }
    } else if (typeof config.params === 'object') {
      Object.values(config.params).forEach(value => {
        if (typeof value === 'string') {
          extractFromText(value).forEach(v => variables.add(v))
        }
      })
    }
  }

  // Extract from workflow definition (业务流程)
  if (config.type === 'workflow' && config.body) {
    try {
      const workflow = typeof config.body === 'string' ? JSON.parse(config.body) : config.body

      // Extract from workflow-level variables
      if (workflow.variables && typeof workflow.variables === 'object') {
        Object.values(workflow.variables).forEach(value => {
          if (typeof value === 'string') {
            extractFromText(value).forEach(v => variables.add(v))
          }
        })
      }

      // Extract from workflow steps
      if (workflow.steps && Array.isArray(workflow.steps)) {
        workflow.steps.forEach((step: any) => {
          // Step URL
          if (step.url) {
            extractFromText(step.url).forEach(v => variables.add(v))
          }

          // Step body
          if (step.body) {
            extractFromText(step.body).forEach(v => variables.add(v))
          }

          // Step proxyUrl
          if (step.proxyUrl) {
            extractFromText(step.proxyUrl).forEach(v => variables.add(v))
          }

          // Step wsMessage
          if (step.wsMessage) {
            extractFromText(step.wsMessage).forEach(v => variables.add(v))
          }

          // Step headers
          if (step.headers && typeof step.headers === 'object') {
            Object.values(step.headers).forEach(value => {
              if (typeof value === 'string') {
                extractFromText(value).forEach(v => variables.add(v))
              }
            })
          }

          // Step params
          if (step.params && typeof step.params === 'object') {
            Object.values(step.params).forEach(value => {
              if (typeof value === 'string') {
                extractFromText(value).forEach(v => variables.add(v))
              }
            })
          }

          // Step assertions (value field may contain variables)
          if (step.assertions && Array.isArray(step.assertions)) {
            step.assertions.forEach((assertion: any) => {
              if (assertion.value) {
                extractFromText(assertion.value).forEach(v => variables.add(v))
              }
            })
          }
        })
      }
    } catch (e) {
      console.warn('Failed to parse workflow definition:', e)
    }
  }

  // Extract from assertions (non-workflow probes)
  if (config.assertions) {
    try {
      const assertions = typeof config.assertions === 'string' ? JSON.parse(config.assertions) : config.assertions
      if (Array.isArray(assertions)) {
        assertions.forEach((assertion: any) => {
          if (assertion.value) {
            extractFromText(assertion.value).forEach(v => variables.add(v))
          }
        })
      }
    } catch {
      // Ignore parse errors
    }
  }

  return Array.from(variables).sort()
}

/**
 * Extract variables from inspection item
 */
export function extractFromInspectionItem(item: any): string[] {
  const variables = new Set<string>()

  if (!item) {
    return []
  }

  // Extract based on execution type
  switch (item.executionType) {
    case 'command':
      if (item.command) {
        extractFromText(item.command).forEach(v => variables.add(v))
      }
      break

    case 'script':
      if (item.scriptContent) {
        extractFromText(item.scriptContent).forEach(v => variables.add(v))
      }
      if (item.scriptArgs) {
        extractFromText(item.scriptArgs).forEach(v => variables.add(v))
      }
      break

    case 'promql':
      if (item.promqlQuery) {
        extractFromText(item.promqlQuery).forEach(v => variables.add(v))
      }
      break

    case 'probe':
      // For probe type inspection items, extract from probe config
      if (item.probeConfigId && item.probeConfig) {
        extractFromProbeConfig(item.probeConfig).forEach(v => variables.add(v))
      }
      break
  }

  return Array.from(variables).sort()
}

/**
 * Extract variables from multiple probe configs
 */
export function extractFromProbeConfigs(configs: any[]): string[] {
  const variables = new Set<string>()

  if (!configs || !Array.isArray(configs)) {
    return []
  }

  configs.forEach(config => {
    extractFromProbeConfig(config).forEach(v => variables.add(v))
  })

  return Array.from(variables).sort()
}

/**
 * Extract variables from multiple inspection items
 */
export function extractFromInspectionItems(items: any[]): string[] {
  const variables = new Set<string>()

  if (!items || !Array.isArray(items)) {
    return []
  }

  items.forEach(item => {
    extractFromInspectionItem(item).forEach(v => variables.add(v))
  })

  return Array.from(variables).sort()
}

/**
 * Convert extracted variable names to customVariablesList format
 */
export function toCustomVariablesList(varNames: string[]): Array<{ key: string; value: string }> {
  return varNames.map(name => ({
    key: name,
    value: '' // Empty value, user needs to fill in
  }))
}

/**
 * Merge extracted variables with existing variables
 * Preserves existing values, only adds new variables
 */
export function mergeVariables(
  existing: Array<{ key: string; value: string }>,
  extracted: string[]
): Array<{ key: string; value: string }> {
  const existingKeys = new Set(existing.map(v => v.key))
  const newVariables = extracted
    .filter(name => !existingKeys.has(name))
    .map(name => ({ key: name, value: '' }))

  return [...existing, ...newVariables]
}
