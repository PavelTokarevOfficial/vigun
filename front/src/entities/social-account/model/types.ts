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
