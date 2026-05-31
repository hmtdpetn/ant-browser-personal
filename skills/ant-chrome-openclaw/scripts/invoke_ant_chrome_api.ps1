param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("GET", "POST", "PUT", "DELETE")]
    [string]$Method,

    [Parameter(Mandatory = $true)]
    [string]$Path,

    [string]$BaseUrl = "",
    [string]$ApiHeader = "",
    [string]$ApiKey = "",
    [string]$JsonBody = "",
    [string]$JsonFile = "",
    [int]$TimeoutSeconds = 15
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Resolve-DefaultValue {
    param(
        [string]$Value,
        [string]$Fallback
    )

    if ([string]::IsNullOrWhiteSpace($Value)) {
        return $Fallback
    }
    return $Value.Trim()
}

function Resolve-RequestUri {
    param(
        [string]$CurrentBaseUrl,
        [string]$CurrentPath
    )

    $normalizedPath = $CurrentPath.Trim()
    if ([Uri]::IsWellFormedUriString($normalizedPath, [UriKind]::Absolute)) {
        return [Uri]$normalizedPath
    }

    if ([string]::IsNullOrWhiteSpace($CurrentBaseUrl)) {
        throw "BaseUrl is required when Path is not an absolute URL."
    }

    $baseText = $CurrentBaseUrl.TrimEnd("/")
    if (-not $normalizedPath.StartsWith("/")) {
        $normalizedPath = "/" + $normalizedPath
    }

    return [Uri]("$baseText$normalizedPath")
}

function Read-RequestBody {
    param(
        [string]$BodyText,
        [string]$BodyFile
    )

    if (-not [string]::IsNullOrWhiteSpace($BodyText) -and -not [string]::IsNullOrWhiteSpace($BodyFile)) {
        throw "JsonBody and JsonFile cannot be used together."
    }

    if (-not [string]::IsNullOrWhiteSpace($BodyFile)) {
        return [System.IO.File]::ReadAllText((Resolve-Path $BodyFile))
    }

    return $BodyText
}

function Try-DecodeJson {
    param(
        [string]$Text
    )

    if ([string]::IsNullOrWhiteSpace($Text)) {
        return $null
    }

    try {
        return $Text | ConvertFrom-Json
    } catch {
        return $null
    }
}

$effectiveBaseUrl = Resolve-DefaultValue $BaseUrl "http://127.0.0.1:19876"
$effectiveApiHeader = Resolve-DefaultValue $ApiHeader "X-Ant-Api-Key"
$effectiveApiKey = Resolve-DefaultValue $ApiKey $env:ANT_CHROME_API_KEY
if ([string]::IsNullOrWhiteSpace($ApiHeader) -and -not [string]::IsNullOrWhiteSpace($env:ANT_CHROME_API_HEADER)) {
    $effectiveApiHeader = $env:ANT_CHROME_API_HEADER.Trim()
}
if ([string]::IsNullOrWhiteSpace($BaseUrl) -and -not [string]::IsNullOrWhiteSpace($env:ANT_CHROME_BASE_URL)) {
    $effectiveBaseUrl = $env:ANT_CHROME_BASE_URL.Trim()
}

$requestUri = Resolve-RequestUri -CurrentBaseUrl $effectiveBaseUrl -CurrentPath $Path
$requestBody = Read-RequestBody -BodyText $JsonBody -BodyFile $JsonFile

$handler = [System.Net.Http.HttpClientHandler]::new()
$client = [System.Net.Http.HttpClient]::new($handler)
$client.Timeout = [TimeSpan]::FromSeconds([Math]::Max(1, $TimeoutSeconds))

$request = [System.Net.Http.HttpRequestMessage]::new([System.Net.Http.HttpMethod]::$Method, $requestUri)
$request.Headers.Accept.Add([System.Net.Http.Headers.MediaTypeWithQualityHeaderValue]::new("application/json"))

if (-not [string]::IsNullOrWhiteSpace($effectiveApiKey)) {
    $request.Headers.TryAddWithoutValidation($effectiveApiHeader, $effectiveApiKey) | Out-Null
}

if (-not [string]::IsNullOrWhiteSpace($requestBody)) {
    $request.Content = [System.Net.Http.StringContent]::new($requestBody, [System.Text.Encoding]::UTF8, "application/json")
}

try {
    $response = $client.SendAsync($request).GetAwaiter().GetResult()
    $responseText = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    $decodedBody = Try-DecodeJson -Text $responseText

    $result = [ordered]@{
        method    = $Method
        url       = $requestUri.AbsoluteUri
        status    = [int]$response.StatusCode
        httpOk    = [bool]$response.IsSuccessStatusCode
        requested = [datetime]::UtcNow.ToString("o")
    }

    if ($null -ne $decodedBody) {
        $result["body"] = $decodedBody
        if ($decodedBody.PSObject.Properties.Name -contains "ok") {
            $result["apiOk"] = [bool]$decodedBody.ok
        }
    } elseif (-not [string]::IsNullOrWhiteSpace($responseText)) {
        $result["rawBody"] = $responseText
    } else {
        $result["rawBody"] = ""
    }

    $result | ConvertTo-Json -Depth 20
    exit 0
} catch {
    $errorResult = [ordered]@{
        method    = $Method
        url       = $requestUri.AbsoluteUri
        status    = 0
        httpOk    = $false
        requested = [datetime]::UtcNow.ToString("o")
        error     = $_.Exception.Message
    }
    $errorResult | ConvertTo-Json -Depth 10
    exit 1
} finally {
    if ($null -ne $request) {
        $request.Dispose()
    }
    if ($null -ne $client) {
        $client.Dispose()
    }
    if ($null -ne $handler) {
        $handler.Dispose()
    }
}
