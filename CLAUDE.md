# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Curator is a personal intelligence platform designed for thought leaders and emerging influencers. It transforms scattered information into structured intelligence by:

- Aggregating cutting-edge discussions from multiple sources before mainstream coverage
- Applying AI-powered quality filtering to surface genuine insights  
- Delivering structured intelligence in both human and AI-readable formats
- Operating entirely under user control with self-hosted infrastructure

## Architecture Vision

The platform is designed around these core components:

1. **Source Adapters**: Pluggable connectors for different platforms/APIs
2. **Content Pipeline**: Ingestion → Processing → Quality Assessment → Storage
3. **LLM Service Layer**: Abstraction over local and remote model endpoints
4. **Curation Engine**: Rules engine + ML models for content scoring
5. **Delivery System**: Template-based output generation and distribution
6. **Management Interface**: Web UI for pipeline configuration and monitoring

## Technical Stack (Planned)

- **Deployment**: Docker-based self-hosted architecture
- **LLM Integration**: Ollama/LocalAI support for local model processing
- **Primary Sources**: Reddit, RSS feeds (MVP), expanding to Twitter/X and specialized forums
- **Output Formats**: Email digests, JSON feeds, knowledge graphs
- **Configuration**: CLI-based pipeline configuration using YAML

## Development Philosophy

- **User Sovereignty**: Users own their data, algorithms, and curation rules
- **Privacy by Design**: Self-hosted architecture eliminates data harvesting
- **Quality over Quantity**: Ruthless filtering for substance over volume
- **Open Foundation**: Built on open-weight models to avoid vendor lock-in
- **Transparency**: Users understand and can modify content filtering and ranking

## MVP Target (8 weeks)

- CLI-based pipeline configuration
- Reddit and RSS source connectors  
- Basic LLM quality scoring via local models
- Email digest output (Markdown → HTML)
- Docker deployment package
- Simple web dashboard for monitoring

## Current Status

This is a very early-stage project currently in the planning phase. The `/planning/` directory contains the foundational product document that outlines the complete vision and roadmap.