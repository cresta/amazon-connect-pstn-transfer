"""
Tests for HTTP client matching Go and TypeScript test structure
"""

from src.httpclient import exponential_backoff, is_retryable_error


class TestIsRetryableError:
    """Tests for is_retryable_error"""

    def test_returns_true_for_error(self):
        """Should return True when error is provided"""
        result = is_retryable_error(Exception("network error"), 0)
        assert result is True

    def test_returns_true_for_500(self):
        """Should return True for 500 status code"""
        result = is_retryable_error(None, 500)
        assert result is True

    def test_returns_true_for_503(self):
        """Should return True for 503 status code"""
        result = is_retryable_error(None, 503)
        assert result is True

    def test_returns_true_for_429(self):
        """Should return True for 429 (Too Many Requests)"""
        result = is_retryable_error(None, 429)
        assert result is True

    def test_returns_true_for_408(self):
        """Should return True for 408 (Request Timeout)"""
        result = is_retryable_error(None, 408)
        assert result is True

    def test_returns_false_for_400(self):
        """Should return False for 400 (Bad Request)"""
        result = is_retryable_error(None, 400)
        assert result is False

    def test_returns_false_for_401(self):
        """Should return False for 401 (Unauthorized)"""
        result = is_retryable_error(None, 401)
        assert result is False

    def test_returns_false_for_404(self):
        """Should return False for 404 (Not Found)"""
        result = is_retryable_error(None, 404)
        assert result is False

    def test_returns_false_for_200(self):
        """Should return False for 200 (OK)"""
        result = is_retryable_error(None, 200)
        assert result is False


class TestExponentialBackoff:
    """Tests for exponential_backoff"""

    def test_first_attempt_returns_base_delay_plus_jitter(self):
        """Should return approximately base delay for first attempt"""
        result = exponential_backoff(0, 100)
        # Base delay is 100ms, jitter adds 0-25%
        assert 100 <= result <= 125

    def test_second_attempt_doubles_delay(self):
        """Should double delay for second attempt"""
        result = exponential_backoff(1, 100)
        # 2^1 * 100 = 200ms, plus 0-25% jitter
        assert 200 <= result <= 250

    def test_third_attempt_quadruples_delay(self):
        """Should quadruple delay for third attempt"""
        result = exponential_backoff(2, 100)
        # 2^2 * 100 = 400ms, plus 0-25% jitter
        assert 400 <= result <= 500

    def test_uses_provided_base_delay(self):
        """Should use provided base delay"""
        result = exponential_backoff(0, 50)
        # Base delay is 50ms, plus 0-25% jitter
        assert 50 <= result <= 62  # 50 + 12.5 max jitter
