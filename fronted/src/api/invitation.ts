import request from '@/api/request'

function wsID(): string {
  return localStorage.getItem('active_ws_id') || ''
}

export interface InvitationVo {
  id: string
  workspaceId: string
  inviterId: string
  email: string
  role: 'admin' | 'editor' | 'member' | 'viewer'
  token: string
  status: 'pending' | 'accepted' | 'expired' | 'revoked'
  expiresAt: string
  createdAt: string
}

export const listInvitations = (params?: any) =>
  request.get(`/workspaces/${wsID()}/invitations`, params)

export const createInvitation = (data: { email: string; role: string }) =>
  request.post(`/workspaces/${wsID()}/invitations`, data)

export const revokeInvitation = (invID: string) =>
  request.delete(`/workspaces/${wsID()}/invitations/${invID}`)
