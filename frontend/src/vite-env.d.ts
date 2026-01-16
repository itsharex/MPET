/// <reference types="vite/client" />

// 声明 SVG 模块
declare module '*.svg' {
  const content: string
  export default content
}

declare module '*.svg?url' {
  const content: string
  export default content
}

declare module '*.svg?raw' {
  const content: string
  export default content
}
