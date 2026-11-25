
export interface TripFlight {
  id: number
  flight_key: string
  origin: string
  destination: string
  origin_airport: string
  destination_airport: string
  departure_at: string
  price: number
  currency: string
  airline: string
  flight_number: string
  transfers: number
  duration: number
  link: string
  booking_link: string
}

export interface TripOption {
  outbound: TripFlight
  inbound: TripFlight
  total_price: number
  currency: string
  trip_duration: number
}

export interface SearchTripsRequest {
  origin: string
  destinations: string[]
  return_origins: string[]
  departure_from: string
  departure_to: string
  min_trip_duration: number
  max_trip_duration: number
  max_transfers: number
  page: number
  limit: number
}

export interface SearchTripsResponse {
  items: TripOption[]
  page: number
  limit: number
}
