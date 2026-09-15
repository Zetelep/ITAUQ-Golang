-- WARNING: This schema is for context only and is not meant to be run.
-- Table order and constraints may not be valid for execution.

CREATE TABLE public.profiles (
  id uuid NOT NULL,
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  full_name text,
  occupation text,
  institution text,
  roles USER-DEFINED NOT NULL DEFAULT 'administrator'::user_role,
  application_id uuid UNIQUE,
  must_change_password boolean NOT NULL DEFAULT true,
  is_active boolean DEFAULT true,
  CONSTRAINT profiles_pkey PRIMARY KEY (id),
  CONSTRAINT profiles_id_fkey FOREIGN KEY (id) REFERENCES auth.users(id),
  CONSTRAINT profiles_application_id_fkey FOREIGN KEY (application_id) REFERENCES public.administrator_applications(id)
);
CREATE TABLE public.questionnaires (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  administrator_id uuid NOT NULL,
  title text NOT NULL,
  app_name text NOT NULL,
  description text,
  itauq_version text NOT NULL DEFAULT 'itauq-v1'::text,
  status USER-DEFINED NOT NULL DEFAULT 'draft'::questionnaire_status,
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  updated_at timestamp with time zone NOT NULL DEFAULT now(),
  app_link text,
  img_link text,
  CONSTRAINT questionnaires_pkey PRIMARY KEY (id),
  CONSTRAINT questionnaires_administrator_id_fkey FOREIGN KEY (administrator_id) REFERENCES public.profiles(id)
);
CREATE TABLE public.task_scenarios (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  questionnaire_id uuid NOT NULL,
  title text NOT NULL,
  instruction text NOT NULL,
  task_order integer NOT NULL DEFAULT 1,
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT task_scenarios_pkey PRIMARY KEY (id),
  CONSTRAINT task_scenarios_questionnaire_id_fkey FOREIGN KEY (questionnaire_id) REFERENCES public.questionnaires(id)
);
CREATE TABLE public.evaluation_links (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  questionnaire_id uuid NOT NULL,
  token text NOT NULL UNIQUE,
  created_by uuid NOT NULL,
  is_active boolean NOT NULL DEFAULT true,
  expires_at timestamp with time zone,
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT evaluation_links_pkey PRIMARY KEY (id),
  CONSTRAINT evaluation_links_questionnaire_id_fkey FOREIGN KEY (questionnaire_id) REFERENCES public.questionnaires(id),
  CONSTRAINT evaluation_links_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.profiles(id)
);
CREATE TABLE public.respondents (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  evaluation_link_id uuid NOT NULL,
  name text NOT NULL,
  email text,
  age integer,
  gender USER-DEFINED,
  occupation text,
  extra_data jsonb NOT NULL DEFAULT '{}'::jsonb,
  started_at timestamp with time zone NOT NULL DEFAULT now(),
  submitted_at timestamp with time zone,
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT respondents_pkey PRIMARY KEY (id),
  CONSTRAINT respondents_evaluation_link_id_fkey FOREIGN KEY (evaluation_link_id) REFERENCES public.evaluation_links(id)
);
CREATE TABLE public.task_scenario_attempts (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  respondent_id uuid NOT NULL,
  task_scenario_id uuid NOT NULL,
  is_success boolean,
  duration_seconds integer,
  notes text,
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT task_scenario_attempts_pkey PRIMARY KEY (id),
  CONSTRAINT task_scenario_attempts_respondent_id_fkey FOREIGN KEY (respondent_id) REFERENCES public.respondents(id),
  CONSTRAINT task_scenario_attempts_task_scenario_id_fkey FOREIGN KEY (task_scenario_id) REFERENCES public.task_scenarios(id)
);
CREATE TABLE public.questionnaire_answers (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  respondent_id uuid NOT NULL,
  item_id integer NOT NULL CHECK (item_id >= 1 AND item_id <= 30),
  category text NOT NULL,
  score integer NOT NULL CHECK (score >= 1 AND score <= 7),
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT questionnaire_answers_pkey PRIMARY KEY (id),
  CONSTRAINT questionnaire_answers_respondent_id_fkey FOREIGN KEY (respondent_id) REFERENCES public.respondents(id)
);
CREATE TABLE public.administrator_applications (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  full_name text NOT NULL,
  email text NOT NULL,
  institution text,
  occupation text,
  reason text,
  status USER-DEFINED NOT NULL DEFAULT 'pending'::application_status,
  reviewed_by uuid,
  reviewed_at timestamp with time zone,
  review_note text,
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT administrator_applications_pkey PRIMARY KEY (id),
  CONSTRAINT administrator_applications_reviewed_by_fkey FOREIGN KEY (reviewed_by) REFERENCES public.profiles(id)
);
CREATE TABLE public.sus_answers (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  respondent_id uuid NOT NULL,
  item_id integer NOT NULL CHECK (item_id >= 1 AND item_id <= 10),
  score integer NOT NULL CHECK (score >= 1 AND score <= 5),
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT sus_answers_pkey PRIMARY KEY (id),
  CONSTRAINT sus_answers_respondent_id_fkey FOREIGN KEY (respondent_id) REFERENCES public.respondents(id)
);
CREATE TABLE public.questionnaire_eligibility_criteria (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  questionnaire_id uuid NOT NULL,
  statement text NOT NULL,
  criteria_order integer NOT NULL DEFAULT 1,
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT questionnaire_eligibility_criteria_pkey PRIMARY KEY (id),
  CONSTRAINT questionnaire_eligibility_criteria_questionnaire_id_fkey FOREIGN KEY (questionnaire_id) REFERENCES public.questionnaires(id)
);
CREATE TABLE public.respondent_eligibility_confirmations (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  respondent_id uuid NOT NULL,
  criteria_id uuid NOT NULL,
  is_checked boolean NOT NULL DEFAULT true,
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT respondent_eligibility_confirmations_pkey PRIMARY KEY (id),
  CONSTRAINT respondent_eligibility_confirmations_respondent_id_fkey FOREIGN KEY (respondent_id) REFERENCES public.respondents(id),
  CONSTRAINT respondent_eligibility_confirmations_criteria_id_fkey FOREIGN KEY (criteria_id) REFERENCES public.questionnaire_eligibility_criteria(id)
);