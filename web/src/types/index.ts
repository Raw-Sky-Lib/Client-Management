// ─── CMS types (client's Supabase — read/written by portal frontend) ─────────

export interface SiteSettings {
  id: string
  key: string
  value: string
  updated_at: string
}

export interface Page {
  id: string
  slug: string
  title: string
  sections: Record<string, unknown>
  seo_title: string | null
  seo_description: string | null
  is_published: boolean
  updated_at: string
}

export interface Post {
  id: string
  slug: string
  title: string
  content: string
  excerpt: string | null
  cover_image_url: string | null
  author_name: string | null
  is_published: boolean
  published_at: string | null
  created_at: string
  updated_at: string
}

export interface NavItem {
  id: string
  label: string
  url: string
  order: number
  is_external: boolean
}

export interface FormSubmission {
  id: string
  form_name: string
  data: Record<string, unknown>
  is_read: boolean
  submitted_at: string
}

export interface Media {
  id: string
  filename: string
  url: string
  mime_type: string
  size_bytes: number
  uploaded_at: string
}

// ─── Claude assistant types ───────────────────────────────────────────────────

export interface FieldChange {
  field: string
  current: string
  proposed: string
  notes: string
}

export interface GenerateRequest {
  page_slug: string
  section_type: string
  instruction: string
}

export interface GenerateResponse {
  changes: FieldChange[]
}

// ─── Section content shapes (pages JSONB) ────────────────────────────────────

export interface HeroSection {
  headline: string
  subheadline: string
  cta_label: string
  cta_url: string
}

export interface FeaturesItem {
  icon: string
  title: string
  description: string
}

export interface FeaturesSection {
  items: FeaturesItem[]
}

export interface AboutSection {
  body: string
  image_url?: string
}

export interface TestimonialItem {
  quote: string
  author: string
  role: string
  avatar?: string
}

export interface TestimonialsSection {
  items: TestimonialItem[]
}

export interface CTASection {
  headline: string
  subheadline: string
  button_label: string
  button_url: string
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

export interface PortalUser {
  user_id: string
  tenant_id: string
  email: string
  supabase_url: string
  supabase_anon_key: string
}
