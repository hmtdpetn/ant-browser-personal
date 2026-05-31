import { getBindings } from './runtime'

export async function openProjectRoot(): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.OpenProjectRoot) {
    await bindings.OpenProjectRoot()
    return true
  }
  return false
}
