"""Job consumer entry: python -m momentos_ml.worker"""

from momentos_ml.worker.loop import run


def main() -> None:
    run()


if __name__ == "__main__":
    main()
