export type TwitchClip = {
  id: string
  title: string
  creator_name?: string
  url: string
  thumbnail_url: string
  duration: number
  created_at: string
  saved: boolean
  viewed?: boolean
}

export type SubscriptionFeed = {
  streamerId: string
  twitchLogin: string
  displayName: string
  lastSyncedAt: string | null
  clips: TwitchClip[]
}
