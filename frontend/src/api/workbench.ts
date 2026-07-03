import { apiClient } from './client'

export interface WorkbenchSSOTicketResponse {
  ticket: string
  expires_in: number
  entry_url: string
}

export const workbenchAPI = {
  createSSOTicket(audience: string): Promise<WorkbenchSSOTicketResponse> {
    return apiClient
      .post<WorkbenchSSOTicketResponse>('/workbench/sso-ticket', { audience })
      .then((response) => response.data)
  }
}
