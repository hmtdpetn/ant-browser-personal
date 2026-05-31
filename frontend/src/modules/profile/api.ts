import profilePageConfig from '../../config/profile.config'
import type { ProfilePageData } from './types'

export function createDefaultProfilePageData(): ProfilePageData {
  return {
    author: {
      ...profilePageConfig.defaultAuthor,
      skills: [...profilePageConfig.defaultAuthor.skills],
      channels: profilePageConfig.defaultAuthor.channels.map((channel) => ({ ...channel })),
    },
    project: {
      ...profilePageConfig.project,
      techStack: [...profilePageConfig.project.techStack],
      actions: profilePageConfig.project.actions.map((action) => ({ ...action })),
    },
    meta: {
      source: 'default',
    },
  }
}

export async function loadProfilePageData(): Promise<ProfilePageData> {
  return createDefaultProfilePageData()
}
