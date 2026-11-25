
import type {
  SearchTripsRequest,
  SearchTripsResponse,
} from '../types/trip'

export async function searchTrips(
  params: SearchTripsRequest,
): Promise<SearchTripsResponse> {
  const response = await fetch('/api/v1/trips/search', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(params),
  })

  if (!response.ok) {
    const message = await response.text()
    throw new Error(message || `HTTP error: ${response.status}`)
  }

  return response.json()
}
