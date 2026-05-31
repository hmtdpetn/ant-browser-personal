import { projectConfig } from './project.config'

export type ProfileIconKey =
  | 'book-open'
  | 'globe'
  | 'message-square'
  | 'github'
  | 'mail'
  | 'external-link'

export interface ProfileChannelConfig {
  name: string
  description: string
  detail: string
  href?: string
  icon?: ProfileIconKey
}

export interface AuthorProfileConfig {
  name: string
  initial: string
  title: string
  bio: string
  location: string
  joinDate: string
  email: string
  website: string
  github: string
  skills: string[]
  channels: ProfileChannelConfig[]
}

export interface ProjectProfileActionConfig {
  label: string
  href: string
  icon: ProfileIconKey
}

export interface ProjectProfileConfig {
  name: string
  introBadge: string
  introText: string
  techStack: string[]
  description: string
  actions: ProjectProfileActionConfig[]
}

export interface RemoteAuthorSourceConfig {
  authorURL: string
  timeoutMs: number
}

export interface ProfilePageLocalConfig {
  remoteAuthor: RemoteAuthorSourceConfig
  defaultAuthor: AuthorProfileConfig
  project: ProjectProfileConfig
}

export const profilePageConfig: ProfilePageLocalConfig = {
  remoteAuthor: {
    authorURL: '',
    timeoutMs: 1000,
  },
  defaultAuthor: {
    name: 'Local Personal Edition',
    initial: 'L',
    title: '',
    bio: '',
    location: '',
    joinDate: '',
    email: '',
    website: '',
    github: '',
    skills: [],
    channels: [],
  },
  project: {
    name: projectConfig.name,
    introBadge: '',
    introText: '',
    techStack: [],
    description: '',
    actions: [],
  },
}

export default profilePageConfig
