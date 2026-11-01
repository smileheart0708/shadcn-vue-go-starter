import { writeAuthToken } from '@/lib/auth/token'
import { ensureMockOwnerSession } from '@/mocks/handlers/session'

export function initializeMockDevSession() {
  writeAuthToken(ensureMockOwnerSession())
}
