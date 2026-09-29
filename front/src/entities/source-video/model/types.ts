export type SourceVideo = {
  id: string
  name: string
  mimeType: string
  size: number
  duration: number
  url: string
  cuts: number
  createdAt: string
}

export type SourceVideoFolder = {
  id: string
  parentId: string | null
  name: string
}

export type SourceVideoLibrary = {
  folderId: string | null
  folders: SourceVideoFolder[]
  videos: SourceVideo[]
}
