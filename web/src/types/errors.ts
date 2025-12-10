/**
 * Error Types
 *
 * Error types for API communication and application-wide error handling
 */

/**
 * API Error
 * Response from the backend
 */
export class ApiError extends Error {
  constructor(
    public statusCode: number,
    public message: string,
    public details?: unknown
  ) {
    super(message);
    this.name = 'ApiError';
    Object.setPrototypeOf(this, ApiError.prototype);
  }

  toJSON() {
    return {
      name: this.name,
      statusCode: this.statusCode,
      message: this.message,
      details: this.details,
    };
  }
}

/**
 * Network Error
 * Communication failed
 */
export class NetworkError extends Error {
  constructor(message: string, public cause?: Error) {
    super(message);
    this.name = 'NetworkError';
    Object.setPrototypeOf(this, NetworkError.prototype);
  }

  toJSON() {
    return {
      name: this.name,
      message: this.message,
      cause: this.cause?.message,
    };
  }
}

/**
 * Validation Error
 * Validation failed for request data
 */
export class ValidationError extends Error {
  constructor(
    public field: string,
    public message: string
  ) {
    super(message);
    this.name = 'ValidationError';
    Object.setPrototypeOf(this, ValidationError.prototype);
  }

  toJSON() {
    return {
      name: this.name,
      field: this.field,
      message: this.message,
    };
  }
}

/**
 * Timeout Error
 * Request timed out
 */
export class TimeoutError extends Error {
  constructor(public timeout: number) {
    super(`Request timed out after ${timeout}ms`);
    this.name = 'TimeoutError';
    Object.setPrototypeOf(this, TimeoutError.prototype);
  }

  toJSON() {
    return {
      name: this.name,
      timeout: this.timeout,
      message: this.message,
    };
  }
}

/**
 * Authentication Error
 * Authentication required or failed
 */
export class AuthenticationError extends Error {
  constructor(message: string = 'Authentication required') {
    super(message);
    this.name = 'AuthenticationError';
    Object.setPrototypeOf(this, AuthenticationError.prototype);
  }

  toJSON() {
    return {
      name: this.name,
      message: this.message,
    };
  }
}

/**
 * Error Response Type (Backend)
 */
export interface ErrorResponse {
  success: false;
  error: string;
  details?: unknown;
  message?: string;
}
