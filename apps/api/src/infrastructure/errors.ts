export class MissingEnvVarError extends Error {
  constructor(name: string) {
    super(`${name} environment variable is required`);
    this.name = 'MissingEnvVarError';
  }
}

export class FrankfurterApiError extends Error {
  constructor(status: number, statusText: string) {
    super(`Frankfurter API error: ${status} ${statusText}`);
    this.name = 'FrankfurterApiError';
  }
}

export class DownstreamFetchError extends Error {
  constructor(url: string, status: number) {
    super(`${url} returned ${status}`);
    this.name = 'DownstreamFetchError';
  }
}
