export type InstagramAccount = {
  id: string
  nickname: string
  instagramUserId: string
  hasAccessToken: boolean
  tokenUpdatedAt: string
  tokenExpiresAt: string | null
  tokenLastCheckedAt: string | null
  verifiedUsername: string
  accountType: string
}

export type InstagramAccountForm = {
  nickname: string
  instagramUserId: string
  accessToken: string
}

export type InstagramReel = {
  id: string
  caption: string
  mediaType: string
  mediaProductType: string
  mediaUrl: string
  thumbnailUrl: string
  permalink: string
  timestamp: string
}

export type InstagramReelPage = {
  items: InstagramReel[]
  nextCursor?: string
}
