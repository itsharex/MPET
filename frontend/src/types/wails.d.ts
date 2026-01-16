declare global {
  interface Window {
    go: {
      main: {
        App: {
          Greet(name: string): Promise<string>
          WindowMinimize(): Promise<void>
          WindowMaximize(): Promise<void>
          WindowUnmaximize(): Promise<void>
          WindowClose(): Promise<void>
          WindowIsMaximized(): Promise<boolean>
        }
      }
    }
  }
}

export {}